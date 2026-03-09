#!/usr/bin/env bash
# ╔══════════════════════════════════════════════════════════════════════════════╗
# ║  HELM OSS AI Audit Orchestrator v2                                         ║
# ║  3-layer pipeline: Mechanical → AI Semantic → Report Merge                 ║
# ║                                                                            ║
# ║  Usage:                                                                    ║
# ║    bash scripts/ci/ai_audit.sh              # all 3 layers                 ║
# ║    bash scripts/ci/ai_audit.sh --mechanical # deterministic only           ║
# ║    bash scripts/ci/ai_audit.sh --ai-only    # Gemini skill only            ║
# ╚══════════════════════════════════════════════════════════════════════════════╝
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
LIB_DIR="$SCRIPT_DIR/lib"
AI_EVIDENCE_DIR="$REPO_ROOT/data/evidence/ai_audit"

RED='\033[0;31m'; GREEN='\033[0;32m'; CYAN='\033[0;36m'; YELLOW='\033[1;33m'
BOLD='\033[1m'; DIM='\033[2m'; NC='\033[0m'
MODE="${1:-all}"

mkdir -p "$AI_EVIDENCE_DIR"

# Source libraries
source "$LIB_DIR/extract-json.sh" 2>/dev/null || true

# ══════════════════════════════════════════════════════════════════════════════
# LAYER 1: Deterministic Checks
# ══════════════════════════════════════════════════════════════════════════════

run_mechanical() {
    echo -e "${BOLD}${CYAN}"
    echo "╔══════════════════════════════════════════════════════════════════════════╗"
    echo "║  Layer 1: Deterministic Audit (22 sections)                       ║"
    echo "╚══════════════════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
    bash "$SCRIPT_DIR/repo_audit.sh"
}

# ══════════════════════════════════════════════════════════════════════════════
# LAYER 2: AI-Powered Semantic Analysis (6 missions)
# ══════════════════════════════════════════════════════════════════════════════

run_ai() {
    echo -e "${BOLD}${CYAN}"
    echo "╔══════════════════════════════════════════════════════════════════════════╗"
    echo "║  Layer 2: AI Semantic Analysis                                         ║"
    echo "╚══════════════════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"

    if ! command -v gemini &>/dev/null; then
        echo -e "  ${RED}Gemini CLI not installed. Skipping AI layer.${NC}"
        return 0
    fi

    local MODEL="${GEMINI_MODEL:-gemini-2.5-pro}"
    local GIT_SHA; GIT_SHA=$(git -C "$REPO_ROOT" rev-parse HEAD 2>/dev/null || echo "unknown")
    echo -e "  ${DIM}Model: $MODEL | Git: ${GIT_SHA:0:8}${NC}"

    # Build manifest
    git -C "$REPO_ROOT" ls-files -- '*.go' | sort > "$AI_EVIDENCE_DIR/manifest.txt"
    local MANIFEST_HASH
    MANIFEST_HASH=$(shasum -a 256 "$AI_EVIDENCE_DIR/manifest.txt" | cut -d' ' -f1)
    local TOTAL_FILES
    TOTAL_FILES=$(wc -l < "$AI_EVIDENCE_DIR/manifest.txt" | tr -d ' ')
    echo -e "  ${DIM}Manifest: $TOTAL_FILES files, sha256:${MANIFEST_HASH:0:16}...${NC}"

    # Write manifest node
    echo "{\"kind\":\"MANIFEST\",\"hash\":\"sha256:${MANIFEST_HASH}\",\"files\":${TOTAL_FILES},\"git_sha\":\"${GIT_SHA}\"}" \
        > "$AI_EVIDENCE_DIR/manifest_node.json"

    # 6 OSS missions
    local MISSIONS=(
        "architecture_coherence:Read ARCHITECTURE.md if present, compare to actual structure in core/pkg"
        "package_completeness:Check core/pkg/* for stubs, empty implementations, and skeleton code"
        "integration_wiring:Find orphan factories/providers/bridges in core/ that are never imported"
        "security_posture:Review core/pkg/auth, crypto, access, trust, guardian, executor for vulnerabilities"
        "doc_code_drift:Check README.md and docs/ against actual code — flag stale references"
        "error_handling:Assess error handling consistency, check for swallowed errors and missing context"
    )

    local MISSIONS_COMPLETED=0
    local MISSIONS_FAILED=0

    for mission_spec in "${MISSIONS[@]}"; do
        local mission_id="${mission_spec%%:*}"
        local mission_hint="${mission_spec#*:}"
        echo ""
        echo -e "  ${BOLD}Mission: $mission_id${NC}"

        local output_file="$AI_EVIDENCE_DIR/${mission_id}.json"
        local prompt="You are auditing the HELM OSS repository at ${REPO_ROOT} (git SHA: ${GIT_SHA}).
This is the OPEN-SOURCE edition — there is NO commercial/ directory.

Mission: ${mission_id}
${mission_hint}

Output ONLY a JSON object with this structure:
{
  \"mission_id\": \"${mission_id}\",
  \"findings\": [
    {\"file\": \"...\", \"category\": \"...\", \"severity\": \"critical|high|medium|low|info\", \"verdict\": \"PASS|FAIL|WARN\", \"title\": \"...\", \"description\": \"...\", \"recommendation\": \"...\"}
  ]
}
No markdown. No fences. Only valid JSON."

        if gemini --help 2>/dev/null | grep -q '\-\-model'; then
            gemini --model "$MODEL" -p "$prompt" > "$output_file" 2>/dev/null || true
        else
            gemini -p "$prompt" > "$output_file" 2>/dev/null || true
        fi

        # Extract JSON if wrapped in markdown
        if [ -f "$output_file" ] && type extract_json &>/dev/null; then
            extract_json "$output_file" "$output_file" 2>/dev/null || true
        fi

        if [ -f "$output_file" ] && python3 -c "import json; json.load(open('$output_file'))" 2>/dev/null; then
            local count
            count=$(python3 -c "import json; d=json.load(open('$output_file')); print(len(d.get('findings',d if isinstance(d,list) else [])))" 2>/dev/null || echo "?")
            echo -e "    ${GREEN}✅${NC} $count findings"
            MISSIONS_COMPLETED=$((MISSIONS_COMPLETED + 1))
        else
            echo -e "    ${YELLOW}⚠️  Output extraction failed${NC}"
            MISSIONS_FAILED=$((MISSIONS_FAILED + 1))
        fi
    done

    echo ""
    echo -e "  ${BOLD}Layer 2 Summary:${NC}"
    echo -e "    Model:     $MODEL"
    echo -e "    Completed: $MISSIONS_COMPLETED / ${#MISSIONS[@]}"
    echo -e "    Failed:    $MISSIONS_FAILED"
}

# ══════════════════════════════════════════════════════════════════════════════
# MAIN
# ══════════════════════════════════════════════════════════════════════════════

case "$MODE" in
    --mechanical)
        run_mechanical
        ;;
    --ai-only)
        run_ai
        ;;
    all|"")
        run_mechanical || true
        echo ""
        run_ai
        ;;
    *)
        echo "Usage: $0 [--mechanical | --ai-only | all]"
        exit 1
        ;;
esac
