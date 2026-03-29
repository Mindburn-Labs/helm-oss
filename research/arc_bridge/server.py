"""HELM ARC-AGI-3 Bridge Sidecar.

Thin FastAPI service wrapping the official ``arc_agi`` Python toolkit.
Exposes a narrow, stable JSON API consumed by the Go ARC Connector.

Supports two modes:
- OFFLINE: local engine (~2000 FPS, no API key, unlimited instances)
- ONLINE:  official REST API (scorecards, replays, 600 RPM limit)

Per ARC-AGI-3 docs, online mode requires cookie-jar session affinity
(AWSALB* cookies) which this bridge handles transparently.

Start:
    uv run uvicorn server:app --port 8787
"""

from __future__ import annotations

import json
import logging
import os
import uuid
from contextlib import asynccontextmanager
from typing import Any

from fastapi import FastAPI, HTTPException, status

from models import (
    CreateSessionRequest,
    GameInfo,
    GameListResponse,
    HealthResponse,
    Observation,
    Frame,
    RunMode,
    ScorecardInfo,
    ScorecardOpenRequest,
    SessionInfo,
    StepRequest,
    StepResponse,
)

logger = logging.getLogger("helm.arc_bridge")
logging.basicConfig(level=logging.INFO, format="%(asctime)s %(name)s %(levelname)s %(message)s")

# ---------------------------------------------------------------------------
# Global state
# ---------------------------------------------------------------------------

_MODE = RunMode(os.getenv("ARC_BRIDGE_MODE", "OFFLINE"))
_sessions: dict[str, "_SessionState"] = {}
_arcade: Any = None  # arc_agi.Arcade instance


class _SessionState:
    """Tracks a live game session."""

    __slots__ = ("session_id", "game_id", "env", "step_count", "done", "last_obs")

    def __init__(self, session_id: str, game_id: str, env: Any) -> None:
        self.session_id = session_id
        self.game_id = game_id
        self.env = env
        self.step_count = 0
        self.done = False
        self.last_obs: Observation | None = None


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def _parse_obs(raw_obs: Any) -> Observation:
    """Convert raw arc_agi observation to our Pydantic model.

    The observation format varies between toolkit versions.
    We normalize to our stable Frame-based schema.
    """
    # The arc_agi toolkit returns observations in various formats.
    # We handle the common patterns defensively.
    frames: list[Frame] = []
    available_actions: list[str] = []
    levels_completed = 0
    total_levels = 1
    done = False
    reward = 0.0
    info: dict[str, Any] = {}

    if raw_obs is None:
        # Fallback: empty observation
        frames = [Frame(grid=[[0]], width=1, height=1)]
        return Observation(
            frames=frames,
            available_actions=available_actions,
            levels_completed=levels_completed,
            total_levels=total_levels,
            done=done,
            reward=reward,
            info=info,
        )

    # Handle tuple return from env.step() → (obs, reward, done, truncated, info)
    if isinstance(raw_obs, tuple):
        if len(raw_obs) >= 5:
            obs_data, reward, done, _truncated, info = raw_obs[:5]
        elif len(raw_obs) >= 4:
            obs_data, reward, done, info = raw_obs[:4]
        elif len(raw_obs) >= 2:
            obs_data = raw_obs[0]
            reward = raw_obs[1] if len(raw_obs) > 1 else 0.0
            done = raw_obs[2] if len(raw_obs) > 2 else False
        else:
            obs_data = raw_obs[0]
    else:
        obs_data = raw_obs

    # Convert info to dict safely
    if info and not isinstance(info, dict):
        try:
            info = dict(info)
        except (TypeError, ValueError):
            info = {"raw": str(info)}

    # Extract grid data
    if isinstance(obs_data, dict):
        # Dict-style observation
        if "grid" in obs_data:
            grid = obs_data["grid"]
            h = len(grid)
            w = len(grid[0]) if h > 0 else 0
            frames = [Frame(grid=grid, width=w, height=h)]
        elif "frames" in obs_data:
            for f in obs_data["frames"]:
                grid = f.get("grid", [[0]])
                h = len(grid)
                w = len(grid[0]) if h > 0 else 0
                frames.append(Frame(grid=grid, width=w, height=h))
        if "available_actions" in obs_data:
            available_actions = obs_data["available_actions"]
        if "levels_completed" in obs_data:
            levels_completed = obs_data["levels_completed"]
        if "total_levels" in obs_data:
            total_levels = obs_data["total_levels"]
    elif hasattr(obs_data, "grid"):
        # Object-style observation
        grid = obs_data.grid if isinstance(obs_data.grid, list) else obs_data.grid.tolist()
        h = len(grid)
        w = len(grid[0]) if h > 0 else 0
        frames = [Frame(grid=grid, width=w, height=h)]
        if hasattr(obs_data, "available_actions"):
            available_actions = list(obs_data.available_actions)
    elif hasattr(obs_data, "__array__") or (isinstance(obs_data, list) and isinstance(obs_data[0], list)):
        # Raw numpy array or nested list
        grid = obs_data.tolist() if hasattr(obs_data, "tolist") else obs_data
        h = len(grid)
        w = len(grid[0]) if h > 0 else 0
        frames = [Frame(grid=grid, width=w, height=h)]

    # Fallback if no frames parsed
    if not frames:
        frames = [Frame(grid=[[0]], width=1, height=1)]

    return Observation(
        frames=frames,
        available_actions=available_actions,
        levels_completed=levels_completed,
        total_levels=total_levels,
        done=bool(done),
        reward=float(reward) if reward else 0.0,
        info=info if isinstance(info, dict) else {},
    )


def _get_session(sid: str) -> _SessionState:
    """Retrieve session or raise 404."""
    s = _sessions.get(sid)
    if s is None:
        raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail=f"Session {sid} not found")
    return s


# ---------------------------------------------------------------------------
# Lifespan
# ---------------------------------------------------------------------------


@asynccontextmanager
async def lifespan(_app: FastAPI):
    """Initialize the ARC arcade on startup."""
    global _arcade
    try:
        import arc_agi

        _arcade = arc_agi.Arcade()
        logger.info("ARC Arcade initialized in %s mode", _MODE.value)
    except ImportError:
        logger.warning("arc_agi not installed — running in STUB mode")
        _arcade = None
    except Exception as exc:
        logger.error("Failed to initialize ARC Arcade: %s", exc)
        _arcade = None
    yield
    # Cleanup: close all sessions
    for sid in list(_sessions):
        try:
            del _sessions[sid]
        except Exception:
            pass
    logger.info("ARC Bridge shutdown complete")


# ---------------------------------------------------------------------------
# FastAPI app
# ---------------------------------------------------------------------------

app = FastAPI(
    title="HELM ARC-AGI-3 Bridge",
    version="0.1.0",
    description="Sidecar bridge for HELM ↔ ARC-AGI-3 integration",
    lifespan=lifespan,
)


@app.get("/health", response_model=HealthResponse)
async def health():
    """Liveness probe."""
    return HealthResponse(
        status="ok" if _arcade is not None else "stub",
        mode=_MODE,
        version="0.1.0",
    )


@app.get("/games", response_model=GameListResponse)
async def list_games():
    """List available ARC-AGI-3 game IDs."""
    if _arcade is None:
        # Stub: return demo games
        demos = [GameInfo(game_id="ls20", description="Latent-State 20 (demo)")]
        return GameListResponse(games=demos, count=len(demos))

    try:
        # Official toolkit: arcade.list_games() or similar
        if hasattr(_arcade, "list_games"):
            game_ids = _arcade.list_games()
        elif hasattr(_arcade, "games"):
            game_ids = list(_arcade.games)
        else:
            # Fallback: try to enumerate
            game_ids = ["ls20"]

        games = [GameInfo(game_id=gid) for gid in game_ids]
        return GameListResponse(games=games, count=len(games))
    except Exception as exc:
        logger.error("Failed to list games: %s", exc)
        raise HTTPException(status_code=500, detail=str(exc))


@app.post("/session", response_model=SessionInfo, status_code=status.HTTP_201_CREATED)
async def create_session(req: CreateSessionRequest):
    """Create a new game session (RESET)."""
    sid = str(uuid.uuid4())

    if _arcade is None:
        # Stub mode: return synthetic session
        stub_obs = Observation(
            frames=[Frame(grid=[[0, 0], [0, 0]], width=2, height=2)],
            available_actions=["ACTION1", "ACTION2", "ACTION3", "ACTION4", "ACTION5"],
            levels_completed=0,
            total_levels=3,
        )
        session = _SessionState(sid, req.game_id, None)
        session.last_obs = stub_obs
        _sessions[sid] = session
        return SessionInfo(session_id=sid, game_id=req.game_id, observation=stub_obs)

    try:
        env = _arcade.make(req.game_id, render_mode="terminal")
        raw_obs = env.reset()
        obs = _parse_obs(raw_obs)

        session = _SessionState(sid, req.game_id, env)
        session.last_obs = obs
        _sessions[sid] = session

        logger.info("Session %s created for game %s", sid, req.game_id)
        return SessionInfo(session_id=sid, game_id=req.game_id, observation=obs)
    except Exception as exc:
        logger.error("Failed to create session for %s: %s", req.game_id, exc)
        raise HTTPException(status_code=500, detail=str(exc))


@app.post("/session/{sid}/step", response_model=StepResponse)
async def step(sid: str, req: StepRequest):
    """Execute an action in a session."""
    session = _get_session(sid)

    if session.done:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="Session already done",
        )

    if session.env is None:
        # Stub mode: increment step, eventually mark done
        session.step_count += 1
        stub_obs = Observation(
            frames=[Frame(grid=[[session.step_count % 16, 0], [0, session.step_count % 16]], width=2, height=2)],
            available_actions=["ACTION1", "ACTION2", "ACTION3", "ACTION4", "ACTION5"],
            levels_completed=min(session.step_count // 5, 3),
            total_levels=3,
            done=session.step_count >= 15,
        )
        session.done = stub_obs.done
        session.last_obs = stub_obs
        return StepResponse(
            session_id=sid,
            step_count=session.step_count,
            observation=stub_obs,
            done=session.done,
        )

    try:
        # Map action string to the appropriate game action
        # arc_agi uses GameAction enum or similar
        action = _resolve_action(req.action, session.env)

        # Build kwargs for reasoning blob if present
        kwargs: dict[str, Any] = {}
        if req.reasoning is not None:
            reasoning_json = json.dumps(req.reasoning)
            if len(reasoning_json) <= 16384:  # 16KB limit per ARC docs
                kwargs["reasoning"] = reasoning_json

        raw_obs = session.env.step(action, **kwargs)
        obs = _parse_obs(raw_obs)

        session.step_count += 1
        session.done = obs.done
        session.last_obs = obs

        return StepResponse(
            session_id=sid,
            step_count=session.step_count,
            observation=obs,
            done=session.done,
        )
    except Exception as exc:
        logger.error("Step failed for session %s: %s", sid, exc)
        raise HTTPException(status_code=500, detail=str(exc))


@app.delete("/session/{sid}", status_code=status.HTTP_204_NO_CONTENT)
async def close_session(sid: str):
    """Close a game session."""
    session = _sessions.pop(sid, None)
    if session is None:
        raise HTTPException(status_code=404, detail=f"Session {sid} not found")
    if session.env is not None and hasattr(session.env, "close"):
        try:
            session.env.close()
        except Exception:
            pass
    logger.info("Session %s closed", sid)


# ---------------------------------------------------------------------------
# Scorecard endpoints (online mode only)
# ---------------------------------------------------------------------------


@app.post("/scorecard", response_model=ScorecardInfo, status_code=status.HTTP_201_CREATED)
async def open_scorecard(req: ScorecardOpenRequest):
    """Open an online scorecard. Only functional in ONLINE mode."""
    if _MODE != RunMode.ONLINE:
        # Stub mode: return synthetic scorecard
        return ScorecardInfo(
            card_id=str(uuid.uuid4()),
            status="open",
        )

    if _arcade is None:
        raise HTTPException(status_code=500, detail="Arcade not initialized")

    try:
        if hasattr(_arcade, "open_scorecard"):
            card = _arcade.open_scorecard(req.game_ids)
            return ScorecardInfo(
                card_id=str(getattr(card, "card_id", uuid.uuid4())),
                status="open",
            )
        else:
            raise HTTPException(status_code=501, detail="Scorecard API not available in this toolkit version")
    except HTTPException:
        raise
    except Exception as exc:
        raise HTTPException(status_code=500, detail=str(exc))


@app.get("/scorecard/{card_id}", response_model=ScorecardInfo)
async def get_scorecard(card_id: str):
    """Get scorecard results."""
    if _MODE != RunMode.ONLINE:
        return ScorecardInfo(
            card_id=card_id,
            status="done",
            scores={"ls20": 0.0},
        )

    if _arcade is None or not hasattr(_arcade, "get_scorecard"):
        raise HTTPException(status_code=501, detail="Scorecard API not available")

    try:
        card = _arcade.get_scorecard(card_id)
        return ScorecardInfo(
            card_id=card_id,
            status=str(getattr(card, "status", "unknown")),
            scores=getattr(card, "scores", {}),
            replay_url=getattr(card, "replay_url", None),
        )
    except Exception as exc:
        raise HTTPException(status_code=500, detail=str(exc))


@app.delete("/scorecard/{card_id}", status_code=status.HTTP_204_NO_CONTENT)
async def close_scorecard(card_id: str):
    """Close a scorecard."""
    if _MODE != RunMode.ONLINE:
        return

    if _arcade is None or not hasattr(_arcade, "close_scorecard"):
        raise HTTPException(status_code=501, detail="Scorecard API not available")

    try:
        _arcade.close_scorecard(card_id)
    except Exception as exc:
        raise HTTPException(status_code=500, detail=str(exc))


# ---------------------------------------------------------------------------
# Action resolution
# ---------------------------------------------------------------------------


def _resolve_action(action_str: str, env: Any) -> Any:
    """Resolve action string to the appropriate engine action type.

    ARC-AGI-3 uses GameAction enum from arcengine. We try to resolve
    the string name (e.g. 'ACTION1') to the actual enum value.
    """
    try:
        from arcengine import GameAction

        # Direct enum lookup
        if hasattr(GameAction, action_str):
            return getattr(GameAction, action_str)
        # Try uppercase
        upper = action_str.upper()
        if hasattr(GameAction, upper):
            return getattr(GameAction, upper)
    except ImportError:
        pass

    # Fallback: pass the raw string (some toolkit versions accept strings)
    return action_str
