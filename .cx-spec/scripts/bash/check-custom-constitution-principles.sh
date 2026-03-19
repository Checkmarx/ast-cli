#!/usr/bin/env bash

set -euo pipefail

JSON_MODE=false

for arg in "$@"; do
    case "$arg" in
        --json)
            JSON_MODE=true
            ;;
        --help|-h)
            cat <<'USAGE'
Usage: check-custom-constitution-principles.sh [--json]

Loads active custom constitution principles from constitution heading prefixes.

Output JSON:
  {
    "active_principles": [
      {
        "id": "...",
        "title": "...",
        "content": "..."
      }
    ]
  }
USAGE
            exit 0
            ;;
        *)
            echo "ERROR: Unknown option '$arg'. Use --help for usage information." >&2
            exit 1
            ;;
    esac
done

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

REPO_ROOT="$(get_repo_root)"
CONSTITUTION_FILE="$REPO_ROOT/.cx-spec/memory/constitution.md"

JSON_MODE_ENV="$JSON_MODE" \
CONSTITUTION_FILE_ENV="$CONSTITUTION_FILE" \
python3 - <<'PY'
import json
import os
import re
from pathlib import Path

constitution_file = Path(os.environ["CONSTITUTION_FILE_ENV"])
json_mode = os.environ.get("JSON_MODE_ENV", "false").lower() == "true"

active = []

if constitution_file.is_file():
    content = constitution_file.read_text(encoding="utf-8", errors="ignore")
    pattern = re.compile(r"^\s*#{2,6}\s+\[CP:([A-Za-z0-9_.-]+)\]\s+(.+?)\s*$", re.IGNORECASE | re.MULTILINE)
    matches = list(pattern.finditer(content))
    section_pattern = re.compile(r"(?m)^##\s+")

    seen = set()
    for idx, match in enumerate(matches):
        principle_id = match.group(1).strip()
        if not principle_id or principle_id in seen:
            continue
        seen.add(principle_id)

        start = match.start()
        end = matches[idx + 1].start() if idx + 1 < len(matches) else len(content)
        next_section = section_pattern.search(content, match.end())
        if next_section and next_section.start() < end:
            end = next_section.start()
        block = content[start:end].strip()

        title = match.group(2).strip() or principle_id

        active.append(
            {
                "id": principle_id,
                "title": title,
                "content": block,
            }
        )

payload = {"active_principles": active}

if json_mode:
    print(json.dumps(payload, ensure_ascii=False))
else:
    if active:
        print("Custom constitution principles: loaded")
        for item in active:
            print(f"- {item['id']}: {item['title']}")
    else:
        print("Custom constitution principles: none active")
PY
