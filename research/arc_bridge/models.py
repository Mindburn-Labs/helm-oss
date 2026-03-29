"""Pydantic models for the HELM ARC-AGI-3 Bridge API.

Maps 1:1 to ARC-AGI-3 observation schema:
- Grids: 2D, ≤64×64, values 0-15
- Frames: 1-N per step (multi-frame transitions)
- Actions: game-specific, step-specific available set
"""

from __future__ import annotations

from enum import Enum
from typing import Any

from pydantic import BaseModel, Field


# ---------------------------------------------------------------------------
# Enums
# ---------------------------------------------------------------------------


class RunMode(str, Enum):
    """Bridge execution mode."""

    OFFLINE = "OFFLINE"
    ONLINE = "ONLINE"


# ---------------------------------------------------------------------------
# ARC observation types
# ---------------------------------------------------------------------------


class Frame(BaseModel):
    """Single 2D grid frame from an ARC environment."""

    grid: list[list[int]] = Field(
        ..., description="2D grid, values 0-15, max 64×64"
    )
    width: int = Field(..., ge=1, le=64)
    height: int = Field(..., ge=1, le=64)


class Observation(BaseModel):
    """Full observation returned after a step or reset."""

    frames: list[Frame] = Field(
        ..., min_length=1, description="1-N frames per step"
    )
    available_actions: list[str] = Field(
        ..., description="Game- and step-specific action set"
    )
    levels_completed: int = Field(default=0, ge=0)
    total_levels: int = Field(default=1, ge=1)
    done: bool = False
    reward: float = 0.0
    info: dict[str, Any] = Field(default_factory=dict)


# ---------------------------------------------------------------------------
# Request / Response models
# ---------------------------------------------------------------------------


class CreateSessionRequest(BaseModel):
    """Request to create a new game session."""

    game_id: str = Field(..., description="ARC game identifier, e.g. 'ls20'")
    mode: RunMode = RunMode.OFFLINE


class StepRequest(BaseModel):
    """Request to execute an action in a session."""

    action: str = Field(..., description="Action name, e.g. 'ACTION1'")
    reasoning: dict[str, Any] | None = Field(
        default=None,
        description="Optional reasoning blob ≤16KB, attached to ARC action",
    )


class SessionInfo(BaseModel):
    """Session state returned on create/step."""

    session_id: str
    game_id: str
    step_count: int = 0
    observation: Observation


class StepResponse(BaseModel):
    """Response from executing a step."""

    session_id: str
    step_count: int
    observation: Observation
    done: bool = False


class GameInfo(BaseModel):
    """Summary info about an available game."""

    game_id: str
    description: str = ""


class GameListResponse(BaseModel):
    """List of available games."""

    games: list[GameInfo]
    count: int


class ScorecardOpenRequest(BaseModel):
    """Request to open an online scorecard."""

    game_ids: list[str] = Field(
        ..., min_length=1, description="Games to include in scorecard"
    )


class ScorecardInfo(BaseModel):
    """Scorecard metadata."""

    card_id: str
    status: str = "open"
    scores: dict[str, Any] = Field(default_factory=dict)
    replay_url: str | None = None


class HealthResponse(BaseModel):
    """Health check response."""

    status: str = "ok"
    mode: RunMode = RunMode.OFFLINE
    version: str = "0.1.0"
