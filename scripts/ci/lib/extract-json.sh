#!/usr/bin/env bash
# scripts/ci/lib/extract-json.sh — Pure bash/awk JSON extractor
# Ported from Mindburn v3 (qa/lib/extract-json.sh)
#
# Usage:
#   source scripts/ci/lib/extract-json.sh
#   extract_json input.txt output.json
#
# Strips markdown fences, finds JSON boundaries, validates with jq.
# Returns 0 on success, 1 on failure. No Python dependency.

extract_json() {
    local input="$1"
    local output="${2:-$1}"

    [ -f "$input" ] || return 1
    [ -s "$input" ] || return 1

    # Step 1: Strip markdown code fences (portable sed — no -i)
    local cleaned
    cleaned=$(sed -e 's/^```json[[:space:]]*$//' -e 's/^```[[:space:]]*$//' "$input")

    # Step 2: Extract JSON boundaries using awk
    # Finds the first [ or { and tracks depth to the matching ] or }
    local extracted
    extracted=$(echo "$cleaned" | awk '
    BEGIN { depth=0; started=0; type="" }
    {
        for (i=1; i<=length($0); i++) {
            c = substr($0, i, 1)
            if (!started) {
                if (c == "[") { type="array"; started=1; depth=1; buf=c }
                else if (c == "{") { type="object"; started=1; depth=1; buf=c }
            } else {
                buf = buf c
                if (type == "array") {
                    if (c == "[") depth++
                    else if (c == "]") { depth--; if (depth == 0) { print buf; exit } }
                } else {
                    if (c == "{") depth++
                    else if (c == "}") { depth--; if (depth == 0) { print buf; exit } }
                }
            }
        }
        if (started) buf = buf "\n"
    }')

    if [ -z "$extracted" ]; then
        return 1
    fi

    # Step 3: Validate with jq and write output
    if echo "$extracted" | jq -e . >/dev/null 2>&1; then
        echo "$extracted" | jq '.' > "$output"
        return 0
    fi

    return 1
}
