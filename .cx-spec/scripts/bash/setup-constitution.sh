#!/usr/bin/env bash

set -eo pipefail

JSON_MODE=false

VALIDATE_MODE=false
SCAN_MODE=false

for arg in "$@"; do
    case "$arg" in
        --json)
            JSON_MODE=true
            ;;
        --validate)
            VALIDATE_MODE=true
            ;;
        --scan)
            SCAN_MODE=true
            ;;
        --help|-h)
            echo "Usage: $0 [--json] [--validate] [--scan]"
            echo "  --json      Output results in JSON format"
            echo "  --validate  Validate existing constitution against team inheritance"
            echo "  --scan      Scan project artifacts and suggest constitution enhancements"
            echo "  --help      Show this help message"
            exit 0
            ;;
        *)
            echo "ERROR: Unknown option '$arg'. Use --help for usage information." >&2
            exit 1
            ;;
    esac
done

# Get script directory and load common functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

# Get all paths and variables from common functions
eval $(get_feature_paths)

# Ensure the .cx-spec/memory directory exists
mkdir -p "$REPO_ROOT/.cx-spec/memory"

CONSTITUTION_FILE="$REPO_ROOT/.cx-spec/memory/constitution.md"
TEMPLATE_FILE="$REPO_ROOT/.cx-spec/templates/constitution-template.md"
INJECTED_CUSTOM_PRINCIPLES_JSON="[]"

# Function to load team constitution
load_team_constitution() {
    local team_constitution=""

    # Try to find team constitution in team directives
    if [[ -n "$TEAM_DIRECTIVES" && -d "$TEAM_DIRECTIVES" ]]; then
        # Look for constitution.md in team directives
        local team_const_file="$TEAM_DIRECTIVES/constitution.md"
        if [[ -f "$team_const_file" ]]; then
            team_constitution=$(cat "$team_const_file")
        else
            # Look in context_modules subdirectory
            team_const_file="$TEAM_DIRECTIVES/context_modules/constitution.md"
            if [[ -f "$team_const_file" ]]; then
                team_constitution=$(cat "$team_const_file")
            fi
        fi
    fi

    # If no team constitution found, use default template
    if [[ -z "$team_constitution" ]]; then
        team_constitution="# Project Constitution

## Core Principles

### Principle 1: Quality First
All code must meet quality standards and include appropriate testing.

### Principle 2: Documentation Required
Clear documentation must accompany all significant changes.

### Principle 3: Security by Default
Security considerations must be addressed for all features.

## Governance

**Version**: 1.0.0 | **Ratified**: $(date +%Y-%m-%d) | **Last Amended**: $(date +%Y-%m-%d)

*This constitution was auto-generated from team defaults. Customize as needed for your project.*"
    fi

    echo "$team_constitution"
}

# Function to parse team principles
parse_team_principles() {
    local team_constitution="$1"

    principles=()
    descriptions=()

    # Split team constitution into lines and parse
    in_description=false
    current_description=""

    while IFS= read -r line; do
        # Check for principle header: "1. **Principle Name**"
        if [[ $line =~ ^([0-9]+)\.\ \*\*(.*)\*\*\ *$ ]]; then
            # Save previous principle if exists
            if [[ ${#principles[@]} -gt 0 && -n "$current_description" ]]; then
                descriptions[${#descriptions[@]}]="$current_description"
            fi

            # Start new principle
            principles[${#principles[@]}]="${BASH_REMATCH[2]}"
            current_description=""
            in_description=true
        elif [[ $in_description == true && -n "$line" ]]; then
            # Accumulate description lines
            if [[ -z "$current_description" ]]; then
                current_description="$line"
            else
                current_description="$current_description $line"
            fi
        fi
    done <<< "$team_constitution"

    # Save last principle description
    if [[ ${#principles[@]} -gt 0 && -n "$current_description" ]]; then
        descriptions[${#descriptions[@]}]="$current_description"
    fi
}

# Function to fill constitution template with team inheritance
fill_constitution_template() {
    local team_constitution="$1"
    local template_content="$2"
    local project_name=""

    # Try to extract project name from git or directory
    if [[ -n "$CURRENT_BRANCH" && "$CURRENT_BRANCH" != "main" ]]; then
        project_name="$CURRENT_BRANCH"
    else
        project_name=$(basename "$REPO_ROOT")
    fi

    # Parse team principles
    parse_team_principles "$team_constitution"

    # Set template variables
    local today=$(date +%Y-%m-%d)

    # Replace placeholders in template
    filled_template="$template_content"
    filled_template="${filled_template//\[PROJECT_NAME\]/$project_name}"
    filled_template="${filled_template//\[CONSTITUTION_VERSION\]/1.0.0}"
    filled_template="${filled_template//\[RATIFICATION_DATE\]/$today}"
    filled_template="${filled_template//\[LAST_AMENDED_DATE\]/$today}"

    # Fill principle placeholders
    for i in {1..5}; do
        # Get principle name and description (arrays are 0-indexed)
        local idx=$((i-1))
        local name_value=""
        local desc_value=""

        if [[ $idx -lt ${#principles[@]} ]]; then
            name_value="${principles[$idx]}"
            desc_value="${descriptions[$idx]}"
        fi

        filled_template="${filled_template//\[PRINCIPLE_${i}_NAME\]/$name_value}"
        filled_template="${filled_template//\[PRINCIPLE_${i}_DESCRIPTION\]/$desc_value}"
    done

    # Fill section placeholders with team governance
    filled_template="${filled_template//\[SECTION_2_NAME\]/Additional Constraints}"
    filled_template="${filled_template//\[SECTION_2_CONTENT\]/All team principles must be followed. Constitution supersedes other practices.}"
    filled_template="${filled_template//\[SECTION_3_NAME\]/Development Workflow}"
    filled_template="${filled_template//\[SECTION_3_CONTENT\]/Follow team constitution principles in all development activities.}"
    filled_template="${filled_template//\[GOVERNANCE_RULES\]/All changes must comply with team constitution. Amendments require team approval.}"

    echo "$filled_template"
}

# Function to validate inheritance integrity
validate_inheritance() {
    local team_constitution="$1"
    local project_constitution="$2"

    # Extract core principles from team constitution
    local team_principles=""
    if echo "$team_constitution" | grep -q "^[0-9]\+\. \*\*.*\*\*"; then
        # Numbered list format
        team_principles=$(echo "$team_constitution" | grep "^[0-9]\+\. \*\*.*\*\*" | sed 's/^[0-9]\+\. \*\{2\}\(.*\)\*\{2\}.*/\1/')
    fi

    # Check if project constitution contains team principles
    local missing_principles=""
    for principle in $team_principles; do
        if ! echo "$project_constitution" | grep -qi "$principle"; then
            missing_principles="$missing_principles$principle, "
        fi
    done

    if [[ -n "$missing_principles" ]]; then
        echo "WARNING: Project constitution may be missing some team principles: ${missing_principles%, }"
        echo "Consider ensuring all team principles are represented in your project constitution."
    else
        echo "✓ Inheritance validation passed - all team principles detected in project constitution"
    fi
}

# Function to check for team constitution updates
check_team_updates() {
    local team_constitution="$1"
    local project_constitution="$2"

    # Check if project constitution has inheritance marker
    if echo "$project_constitution" | grep -q "Inherited from team constitution"; then
        local inheritance_date=""
        inheritance_date=$(echo "$project_constitution" | grep "Inherited from team constitution" | sed 's/.*on \([0-9-]\+\).*/\1/')

        if [[ -n "$inheritance_date" ]]; then
            # Get team constitution file modification date
            local team_file=""
            if [[ -n "$TEAM_DIRECTIVES" && -d "$TEAM_DIRECTIVES" ]]; then
                if [[ -f "$TEAM_DIRECTIVES/constitution.md" ]]; then
                    team_file="$TEAM_DIRECTIVES/constitution.md"
                elif [[ -f "$TEAM_DIRECTIVES/context_modules/constitution.md" ]]; then
                    team_file="$TEAM_DIRECTIVES/context_modules/constitution.md"
                fi
            fi

            if [[ -n "$team_file" ]]; then
                local team_mod_date=""
                team_mod_date=$(stat -c %Y "$team_file" 2>/dev/null)

                local inheritance_timestamp=""
                inheritance_timestamp=$(date -d "$inheritance_date" +%s 2>/dev/null)

                if [[ -n "$team_mod_date" && -n "$inheritance_timestamp" && "$team_mod_date" -gt "$inheritance_timestamp" ]]; then
                    echo "NOTICE: Team constitution has been updated since project constitution was created."
                    echo "Consider reviewing the team constitution for any changes that should be reflected in your project."
                    echo "Team constitution: $team_file"
                fi
            fi
        fi
    fi
}

# Inject custom constitution principle snippets based on generic detection.
# Input: constitution markdown via stdin
# Output: JSON {"constitution":"...","injected_principles":["principle_id",...]}
inject_custom_principles_json() {
    local constitution_input_tmp
    local status=0
    constitution_input_tmp=$(mktemp)
    cat > "$constitution_input_tmp"

    if ! REPO_ROOT_ENV="$REPO_ROOT" \
    SCRIPT_DIR_ENV="$SCRIPT_DIR" \
    CUSTOM_CONSTITUTION_PRINCIPLES_CONFIG_ENV="${CUSTOM_CONSTITUTION_PRINCIPLES_CONFIG:-}" \
    python3 - "$constitution_input_tmp" <<'PY'
import json
import os
import re
import sys
from fnmatch import fnmatch
from pathlib import Path

MAX_SCAN_FILE_SIZE_BYTES = 2 * 1024 * 1024


def to_list(value):
    if value is None:
        return []
    if isinstance(value, list):
        return [item for item in value if isinstance(item, str) and item.strip()]
    if isinstance(value, str) and value.strip():
        return [value.strip()]
    return []


def matches_glob(rel_path, pattern):
    pattern = (pattern or "").strip()
    if not pattern:
        return False
    if pattern in ("**", "**/*", "*"):
        return True
    if fnmatch(rel_path, pattern):
        return True
    if "/" not in pattern and fnmatch(Path(rel_path).name, pattern):
        return True
    return False


def path_matches_any(rel_path, patterns):
    if not patterns:
        return True
    return any(matches_glob(rel_path, pattern) for pattern in patterns)


def collect_repo_files(repo_root, include_any, exclude_any, cache):
    key = (tuple(include_any), tuple(exclude_any))
    if key in cache:
        return cache[key]

    files = []
    for path in repo_root.rglob("*"):
        if not path.is_file():
            continue
        rel = path.relative_to(repo_root).as_posix()
        if rel.startswith(".git/"):
            continue
        if not path_matches_any(rel, include_any):
            continue
        if path_matches_any(rel, exclude_any):
            continue
        files.append(path)

    cache[key] = files
    return files


def read_file_text(path, content_cache):
    key = str(path)
    if key in content_cache:
        return content_cache[key]

    try:
        if path.stat().st_size > MAX_SCAN_FILE_SIZE_BYTES:
            content_cache[key] = ""
            return ""
        content = path.read_text(encoding="utf-8", errors="ignore")
    except Exception:
        content = ""

    content_cache[key] = content
    return content


def evaluate_regex_condition(condition, repo_root, cache):
    condition_type = (condition.get("type") or "").strip()
    include_any = to_list(condition.get("include_any"))
    exclude_any = to_list(condition.get("exclude_any"))
    patterns_any = to_list(condition.get("patterns_any"))

    if not include_any:
        include_any = ["**/*"]

    if condition_type not in {"regex_any", "regex_all"}:
        raise ValueError(f"Unsupported detection type '{condition_type}'.")
    if not patterns_any:
        raise ValueError("Detection regex condition must define non-empty patterns_any.")

    compiled = []
    for pattern in patterns_any:
        try:
            compiled.append(re.compile(pattern, re.MULTILINE))
        except re.error as exc:
            raise ValueError(f"Invalid regex '{pattern}': {exc}") from exc

    files = collect_repo_files(repo_root, include_any, exclude_any, cache["glob"])
    if not files:
        return False

    if condition_type == "regex_any":
        for file_path in files:
            content = read_file_text(file_path, cache["content"])
            if not content:
                continue
            for pattern in compiled:
                if pattern.search(content):
                    return True
        return False

    remaining = set(range(len(compiled)))
    for file_path in files:
        if not remaining:
            break
        content = read_file_text(file_path, cache["content"])
        if not content:
            continue
        for idx in list(remaining):
            if compiled[idx].search(content):
                remaining.remove(idx)
    return not remaining


def evaluate_detection(detection, repo_root, cache):
    if not isinstance(detection, dict):
        return False

    rules_any = detection.get("rules_any")
    rules_all = detection.get("rules_all")

    if isinstance(rules_any, list) and rules_any:
        return any(
            evaluate_regex_condition(condition, repo_root, cache)
            for condition in rules_any
            if isinstance(condition, dict)
        )

    if isinstance(rules_all, list) and rules_all:
        return all(
            evaluate_regex_condition(condition, repo_root, cache)
            for condition in rules_all
            if isinstance(condition, dict)
        )

    return False


def resolve_registry_path(repo_root, script_dir):
    candidates = []

    env_config = os.environ.get("CUSTOM_CONSTITUTION_PRINCIPLES_CONFIG_ENV", "").strip()
    if env_config:
        candidate = Path(env_config)
        if not candidate.is_absolute():
            candidate = repo_root / candidate
        candidates.append(candidate)

    project_config = repo_root / ".cx-spec" / "config.json"
    if project_config.is_file():
        try:
            payload = json.loads(project_config.read_text(encoding="utf-8"))
            configured = payload.get("custom_constitution_principles", {}).get("config")
            if isinstance(configured, str) and configured.strip():
                candidate = Path(configured.strip())
                if not candidate.is_absolute():
                    candidate = repo_root / candidate
                candidates.append(candidate)
        except Exception:
            # Ignore malformed config during constitution generation fallback.
            pass

    candidates.append(repo_root / ".cx-spec" / "templates" / "custom-constitution-principles.json")
    candidates.append(script_dir.parent.parent / "templates" / "custom-constitution-principles.json")

    seen = set()
    for candidate in candidates:
        key = str(candidate)
        if key in seen:
            continue
        seen.add(key)
        if candidate.is_file():
            return candidate

    return None


def main():
    constitution_content = Path(sys.argv[1]).read_text(encoding="utf-8", errors="ignore")
    repo_root = Path(os.environ["REPO_ROOT_ENV"])
    script_dir = Path(os.environ["SCRIPT_DIR_ENV"])
    placeholder = "[CUSTOM_CONSTITUTION_PRINCIPLES]"

    result = {
        "constitution": constitution_content,
        "injected_principles": [],
    }

    registry_path = resolve_registry_path(repo_root, script_dir)
    if registry_path is None:
        print(json.dumps(result, ensure_ascii=False))
        return

    payload = json.loads(registry_path.read_text(encoding="utf-8"))
    principles = payload.get("principles")
    if not isinstance(principles, list):
        raise ValueError(f"Invalid principles registry format: {registry_path}")

    cache = {"glob": {}, "content": {}}
    snippets = []

    for principle in principles:
        if not isinstance(principle, dict):
            continue
        if principle.get("enabled", True) is False:
            continue

        principle_file = (principle.get("principle_file") or "").strip()
        if not principle_file:
            continue

        if not evaluate_detection(principle.get("detection"), repo_root, cache):
            continue

        principle_path = Path(principle_file)
        if not principle_path.is_absolute():
            principle_path = registry_path.parent / principle_path
        if not principle_path.is_file():
            raise FileNotFoundError(
                f"Principle file not found for principle '{principle.get('id', '')}': {principle_path}"
            )

        principle_content = principle_path.read_text(encoding="utf-8")
        snippets.append(principle_content.strip())
        result["injected_principles"].append(principle.get("id", ""))

    if snippets:
        section = "## Custom Constitution Principles\n\n" + "\n\n".join(snippets).strip() + "\n\n"
        if placeholder in constitution_content:
            result["constitution"] = constitution_content.replace(placeholder, section.rstrip(), 1)
        elif "## Governance" in constitution_content:
            result["constitution"] = constitution_content.replace("## Governance", section + "## Governance", 1)
        else:
            result["constitution"] = constitution_content.rstrip() + "\n\n" + section.rstrip() + "\n"
    elif placeholder in constitution_content:
        result["constitution"] = constitution_content.replace(placeholder, "", 1)

    print(json.dumps(result, ensure_ascii=False))


if __name__ == "__main__":
    main()
PY
    then
        status=$?
    fi

    rm -f "$constitution_input_tmp"
    return $status
}

# Scan-only mode
if $SCAN_MODE && [[ ! -f "$CONSTITUTION_FILE" ]]; then
    echo "Scanning project artifacts for constitution suggestions..."
    "$SCRIPT_DIR/scan-project-artifacts.sh" --suggestions
    exit 0
fi

# Validation-only mode
if $VALIDATE_MODE; then
    if [[ ! -f "$CONSTITUTION_FILE" ]]; then
        echo "ERROR: No constitution file found at $CONSTITUTION_FILE"
        echo "Run without --validate to create the constitution first."
        exit 1
    fi

    # Load constitutions for validation
    TEAM_CONSTITUTION=$(load_team_constitution)
    PROJECT_CONSTITUTION=$(cat "$CONSTITUTION_FILE")

    if $JSON_MODE; then
        # Basic validation result
        printf '{"status":"validated","file":"%s","team_directives":"%s"}\n' "$CONSTITUTION_FILE" "$TEAM_DIRECTIVES"
    else
        echo "Validating constitution at: $CONSTITUTION_FILE"
        echo "Team directives source: $TEAM_DIRECTIVES"
        echo ""
        validate_inheritance "$TEAM_CONSTITUTION" "$PROJECT_CONSTITUTION"
        echo ""
        check_team_updates "$TEAM_CONSTITUTION" "$PROJECT_CONSTITUTION"
    fi
    exit 0
fi

# Main logic
if [[ -f "$CONSTITUTION_FILE" ]]; then
    echo "Constitution file already exists at $CONSTITUTION_FILE"
    echo "Use git to modify it directly, or remove it to recreate from team directives."

    # Load team constitution for comparison
    TEAM_CONSTITUTION=$(load_team_constitution)
    EXISTING_CONSTITUTION=$(cat "$CONSTITUTION_FILE")

    # Check for team constitution updates
    if ! $JSON_MODE; then
        check_team_updates "$TEAM_CONSTITUTION" "$EXISTING_CONSTITUTION"
        echo ""
    fi

    if $JSON_MODE; then
        printf '{"status":"exists","file":"%s"}\n' "$CONSTITUTION_FILE"
    fi
    exit 0
fi

# Load team constitution
TEAM_CONSTITUTION=$(load_team_constitution)

# Load constitution template
if [[ ! -f "$TEMPLATE_FILE" ]]; then
    echo "ERROR: Constitution template not found at $TEMPLATE_FILE"
    exit 1
fi
TEMPLATE_CONTENT=$(cat "$TEMPLATE_FILE")

# Fill template with team inheritance
PROJECT_CONSTITUTION=$(fill_constitution_template "$TEAM_CONSTITUTION" "$TEMPLATE_CONTENT")

# If scan mode is enabled, enhance constitution with project insights
if $SCAN_MODE; then
    if ! $JSON_MODE; then
        echo "Enhancing constitution with project artifact analysis..."
    fi

    # Get scan suggestions
    SCAN_SUGGESTIONS=$("$SCRIPT_DIR/scan-project-artifacts.sh" --json)

    # Parse scan data and generate suggestions
    TESTING_DATA=$(echo "$SCAN_SUGGESTIONS" | jq -r '.testing')
    SECURITY_DATA=$(echo "$SCAN_SUGGESTIONS" | jq -r '.security')
    DOCS_DATA=$(echo "$SCAN_SUGGESTIONS" | jq -r '.documentation')
    ARCH_DATA=$(echo "$SCAN_SUGGESTIONS" | jq -r '.architecture')

    # Generate additional principles based on scan
    ADDITIONAL_PRINCIPLES=""

    # Parse testing data
    TEST_FILES=$(echo "$TESTING_DATA" | cut -d'|' -f1)
    TEST_FRAMEWORKS=$(echo "$TESTING_DATA" | cut -d'|' -f2)

    if [[ $TEST_FILES -gt 0 ]]; then
        ADDITIONAL_PRINCIPLES="${ADDITIONAL_PRINCIPLES}
### Tests Drive Confidence (Project Practice)
Automated testing is established with $TEST_FILES test files using $TEST_FRAMEWORKS. All features must maintain or improve test coverage. Refuse to ship when test suites fail."
    fi

    # Parse security data
    AUTH_PATTERNS=$(echo "$SECURITY_DATA" | cut -d'|' -f1)
    SECURITY_INDICATORS=$(echo "$SECURITY_DATA" | cut -d'|' -f3)

    if [[ $AUTH_PATTERNS -gt 0 || $SECURITY_INDICATORS -gt 0 ]]; then
        ADDITIONAL_PRINCIPLES="${ADDITIONAL_PRINCIPLES}
### Security by Default (Project Practice)
Security practices are established in the codebase. All features must include security considerations, input validation, and follow established security patterns."
    fi

    # Parse documentation data
    README_COUNT=$(echo "$DOCS_DATA" | cut -d'|' -f1)
    COMMENT_PERCENTAGE=$(echo "$DOCS_DATA" | cut -d'|' -f3)

    if [[ $README_COUNT -gt 0 ]]; then
        ADDITIONAL_PRINCIPLES="${ADDITIONAL_PRINCIPLES}
### Documentation Matters (Project Practice)
Documentation practices are established with $README_COUNT README files. All features must include appropriate documentation and maintain existing documentation standards."
    fi

    # Insert additional principles into constitution
    if [[ -n "$ADDITIONAL_PRINCIPLES" ]]; then
        # Find the end of core principles section and insert additional principles
        PROJECT_CONSTITUTION=$(echo "$PROJECT_CONSTITUTION" | sed "/## Additional Constraints/i\\
## Project-Specific Principles\\
$ADDITIONAL_PRINCIPLES")
    fi
fi

# Inject custom constitution principles based on generic detection.
INJECTION_OUTPUT=$(printf "%s" "$PROJECT_CONSTITUTION" | inject_custom_principles_json)
INJECTION_TMP=$(mktemp)
printf "%s" "$INJECTION_OUTPUT" > "$INJECTION_TMP"
PROJECT_CONSTITUTION=$(python3 - "$INJECTION_TMP" <<'PY'
import json
import sys

payload = json.load(open(sys.argv[1], encoding="utf-8"))
print(payload.get("constitution", ""), end="")
PY
)
INJECTED_CUSTOM_PRINCIPLES_JSON=$(python3 - "$INJECTION_TMP" <<'PY'
import json
import sys

payload = json.load(open(sys.argv[1], encoding="utf-8"))
print(json.dumps(payload.get("injected_principles", []), ensure_ascii=False))
PY
)
rm -f "$INJECTION_TMP"

# Validate inheritance integrity
if ! $JSON_MODE; then
    validate_inheritance "$TEAM_CONSTITUTION" "$PROJECT_CONSTITUTION"
    echo ""
fi

# Write to file
echo "$PROJECT_CONSTITUTION" > "$CONSTITUTION_FILE"

# Output results
if $JSON_MODE; then
    printf '{"status":"created","file":"%s","team_directives":"%s","injected_custom_principles":%s}\n' "$CONSTITUTION_FILE" "$TEAM_DIRECTIVES" "$INJECTED_CUSTOM_PRINCIPLES_JSON"
else
    echo "Constitution created at: $CONSTITUTION_FILE"
    echo "Team directives source: $TEAM_DIRECTIVES"
    if [[ "$INJECTED_CUSTOM_PRINCIPLES_JSON" != "[]" ]]; then
        echo "Injected custom principles: $INJECTED_CUSTOM_PRINCIPLES_JSON"
    fi
    echo ""
    echo "Next steps:"
    echo "1. Review and customize the constitution for your project needs"
    echo "2. Commit the constitution: git add .cx-spec/memory/constitution.md && git commit -m 'docs: initialize project constitution'"
    echo "3. The constitution will be used by planning and implementation commands"
fi
