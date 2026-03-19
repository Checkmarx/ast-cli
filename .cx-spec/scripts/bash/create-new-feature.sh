#!/usr/bin/env bash

set -e

JSON_MODE=false
SHORT_NAME=""
BRANCH_NUMBER=""
MODE="spec"
TDD=""
CONTRACTS=""
DATA_MODELS=""
RISK_TESTS=""
ARGS=()
i=1
while [ $i -le $# ]; do
    arg="${!i}"
    case "$arg" in
        --json) 
            JSON_MODE=true 
            ;;
        --short-name)
            if [ $((i + 1)) -gt $# ]; then
                echo 'Error: --short-name requires a value' >&2
                exit 1
            fi
            i=$((i + 1))
            next_arg="${!i}"
            # Check if the next argument is another option (starts with --)
            if [[ "$next_arg" == --* ]]; then
                echo 'Error: --short-name requires a value' >&2
                exit 1
            fi
            SHORT_NAME="$next_arg"
            ;;
        --number)
            if [ $((i + 1)) -gt $# ]; then
                echo 'Error: --number requires a value' >&2
                exit 1
            fi
            i=$((i + 1))
            next_arg="${!i}"
            if [[ "$next_arg" == --* ]]; then
                echo 'Error: --number requires a value' >&2
                exit 1
            fi
            BRANCH_NUMBER="$next_arg"
            ;;
        --mode)
            if [ $((i + 1)) -gt $# ]; then
                echo 'Error: --mode requires a value' >&2
                exit 1
            fi
            i=$((i + 1))
            next_arg="${!i}"
            if [[ "$next_arg" == --* ]]; then
                echo 'Error: --mode requires a value' >&2
                exit 1
            fi
            MODE="$next_arg"
            ;;
        --tdd)
            if [ $((i + 1)) -gt $# ]; then
                echo 'Error: --tdd requires a value' >&2
                exit 1
            fi
            i=$((i + 1))
            TDD="${!i}"
            ;;
        --contracts)
            if [ $((i + 1)) -gt $# ]; then
                echo 'Error: --contracts requires a value' >&2
                exit 1
            fi
            i=$((i + 1))
            CONTRACTS="${!i}"
            ;;
        --data-models)
            if [ $((i + 1)) -gt $# ]; then
                echo 'Error: --data-models requires a value' >&2
                exit 1
            fi
            i=$((i + 1))
            DATA_MODELS="${!i}"
            ;;
        --risk-tests)
            if [ $((i + 1)) -gt $# ]; then
                echo 'Error: --risk-tests requires a value' >&2
                exit 1
            fi
            i=$((i + 1))
            RISK_TESTS="${!i}"
            ;;
        --help|-h) 
            echo "Usage: $0 [OPTIONS] <feature_description_with_jira>"
            echo ""
            echo "Options:"
            echo "  --json                  Output in JSON format"
            echo "  --short-name <name>     Provide a custom short name (2-4 words) for the branch suffix"
            echo "  --number N              Deprecated (ignored)"
            echo "  --mode <build|spec>     Workflow mode (default: spec)"
            echo "  --tdd <true|false>      Enable TDD (default: mode-specific)"
            echo "  --contracts <true|false> Enable API contracts (default: mode-specific)"
            echo "  --data-models <true|false> Enable data models (default: mode-specific)"
            echo "  --risk-tests <true|false> Enable risk-based testing (default: mode-specific)"
            echo "  --help, -h              Show this help message"
            echo ""
            echo "Mode Defaults:"
            echo "  build: tdd=false, contracts=false, data_models=false, risk_tests=false"
            echo "  spec:  tdd=false, contracts=true, data_models=true, risk_tests=true"
            echo ""
            echo "Examples:"
            echo "  $0 'SCA-123456 Add user authentication system' --short-name 'user-auth'"
            echo "  $0 --mode build 'SCA-123456 Quick feature prototype'"
            echo "  $0 --mode spec --tdd false 'SCA-123456 Feature without TDD' --number 5"
            exit 0
            ;;
        *) 
            ARGS+=("$arg") 
            ;;
    esac
    i=$((i + 1))
done

FEATURE_DESCRIPTION="${ARGS[*]}"
if [ -z "$FEATURE_DESCRIPTION" ]; then
    echo "Usage: $0 [--json] [--short-name <name>] [--number N] <feature_description_with_jira>" >&2
    exit 1
fi

extract_jira_id() {
    local input="$1"
    if [[ "$input" =~ ([A-Za-z][A-Za-z0-9]+-[0-9]+) ]]; then
        echo "${BASH_REMATCH[1]}"
    fi
}

remove_jira_from_description() {
    local input="$1"
    local jira="$2"
    # Remove the first jira occurrence and trim extra spaces.
    echo "$input" | sed -E "s/(^|[[:space:]])${jira}([[:space:]]|$)/ /I" | sed 's/^[[:space:]]*//; s/[[:space:]]*$//' | tr -s ' '
}

validate_jira_id() {
    local jira="$1"
    [[ "$jira" =~ ^[A-Za-z][A-Za-z0-9]+-[0-9]+$ ]]
}

JIRA_ID=$(extract_jira_id "$FEATURE_DESCRIPTION")
if [ -z "$JIRA_ID" ] || ! validate_jira_id "$JIRA_ID"; then
    echo "Error: feature description must include a JIRA ID (example: SCA-123456 New feature)." >&2
    exit 1
fi

FEATURE_DESCRIPTION_NO_JIRA=$(remove_jira_from_description "$FEATURE_DESCRIPTION" "$JIRA_ID")
if [ -z "$FEATURE_DESCRIPTION_NO_JIRA" ]; then
    FEATURE_DESCRIPTION_NO_JIRA="feature"
fi
JIRA_ID_LOWER=$(echo "$JIRA_ID" | tr '[:upper:]' '[:lower:]')

# Function to find the repository root by searching for existing project markers
find_repo_root() {
    local dir="$1"
    while [ "$dir" != "/" ]; do
        if [ -d "$dir/.git" ] || [ -d "$dir/.cx-spec" ]; then
            echo "$dir"
            return 0
        fi
        dir="$(dirname "$dir")"
    done
    return 1
}

# Function to clean and format a branch name
clean_branch_name() {
    local name="$1"
    echo "$name" | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9]/-/g' | sed 's/-\+/-/g' | sed 's/^-//' | sed 's/-$//'
}

# Resolve repository root. Prefer git information when available, but fall back
# to searching for repository markers so the workflow still functions in repositories that
# were initialised with --no-git.
SCRIPT_DIR="$(CDPATH="" cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if git rev-parse --show-toplevel >/dev/null 2>&1; then
    REPO_ROOT=$(git rev-parse --show-toplevel)
    HAS_GIT=true
else
    REPO_ROOT="$(find_repo_root "$SCRIPT_DIR")"
    if [ -z "$REPO_ROOT" ]; then
        echo "Error: Could not determine repository root. Please run this script from within the repository." >&2
        exit 1
    fi
    HAS_GIT=false
fi

cd "$REPO_ROOT"

# Get project-level config path (.cx-spec/config.json)
get_project_config_path() {
    echo "$REPO_ROOT/.cx-spec/config.json"
}

# Get config path (repo-local only)
get_config_path() {
    get_project_config_path
}

get_team_directives_path() {
    command -v jq >/dev/null 2>&1 || { echo ""; return; }

    local project_config
    project_config=$(get_project_config_path)

    local candidates=()
    [[ -f "$project_config" ]] && candidates+=("$project_config")

    local cfg
    for cfg in "${candidates[@]}"; do
        local path
        path=$(jq -r '.team_directives.path // empty' "$cfg" 2>/dev/null)
        if [[ -n "$path" && "$path" != "null" ]]; then
            if [[ "$path" = /* ]]; then
                echo "$path"
            else
                echo "$REPO_ROOT/$path"
            fi
            return
        fi
    done
    echo ""
}

SPECS_DIR="$REPO_ROOT/specs"
mkdir -p "$SPECS_DIR"

# Function to generate branch name with stop word filtering and length filtering
generate_branch_name() {
    local description="$1"
    
    # Common stop words to filter out
    local stop_words="^(i|a|an|the|to|for|of|in|on|at|by|with|from|is|are|was|were|be|been|being|have|has|had|do|does|did|will|would|should|could|can|may|might|must|shall|this|that|these|those|my|your|our|their|want|need|add|get|set)$"
    
    # Convert to lowercase and split into words
    local clean_name=$(echo "$description" | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9]/ /g')
    
    # Filter words: remove stop words and words shorter than 3 chars (unless they're uppercase acronyms in original)
    local meaningful_words=()
    for word in $clean_name; do
        # Skip empty words
        [ -z "$word" ] && continue
        
        # Keep words that are NOT stop words AND (length >= 3 OR are potential acronyms)
        if ! echo "$word" | grep -qiE "$stop_words"; then
            if [ ${#word} -ge 3 ]; then
                meaningful_words+=("$word")
            elif echo "$description" | grep -q "\b${word^^}\b"; then
                # Keep short words if they appear as uppercase in original (likely acronyms)
                meaningful_words+=("$word")
            fi
        fi
    done
    
    # If we have meaningful words, use first 3-4 of them
    if [ ${#meaningful_words[@]} -gt 0 ]; then
        local max_words=3
        if [ ${#meaningful_words[@]} -eq 4 ]; then max_words=4; fi
        
        local result=""
        local count=0
        for word in "${meaningful_words[@]}"; do
            if [ $count -ge $max_words ]; then break; fi
            if [ -n "$result" ]; then result="$result-"; fi
            result="$result$word"
            count=$((count + 1))
        done
        echo "$result"
    else
        # Fallback to original logic if no meaningful words found
        local cleaned=$(clean_branch_name "$description")
        echo "$cleaned" | tr '-' '\n' | grep -v '^$' | head -3 | tr '\n' '-' | sed 's/-$//'
    fi
}

# Generate branch name
if [ -n "$SHORT_NAME" ]; then
    # Use provided short name, just clean it up
    BRANCH_SUFFIX=$(clean_branch_name "$SHORT_NAME")
else
    # Generate from description with smart filtering
    BRANCH_SUFFIX=$(generate_branch_name "$FEATURE_DESCRIPTION_NO_JIRA")
fi

# Build branch from jira id + suffix (legacy --number ignored).
if [ -n "$BRANCH_NUMBER" ]; then
    >&2 echo "[specify] Warning: --number is deprecated and ignored in CX Jira mode."
fi

if [ -z "$BRANCH_SUFFIX" ]; then
    BRANCH_SUFFIX="feature"
fi
if [[ "$BRANCH_SUFFIX" == "${JIRA_ID_LOWER}"-* ]]; then
    BRANCH_SUFFIX="${BRANCH_SUFFIX#${JIRA_ID_LOWER}-}"
fi
if [ "$BRANCH_SUFFIX" = "$JIRA_ID_LOWER" ]; then
    BRANCH_NAME="$JIRA_ID_LOWER"
else
    BRANCH_NAME="${JIRA_ID_LOWER}-${BRANCH_SUFFIX}"
fi

# GitHub enforces a 244-byte limit on branch names
# Validate and truncate if necessary
MAX_BRANCH_LENGTH=244
if [ ${#BRANCH_NAME} -gt $MAX_BRANCH_LENGTH ]; then
    # Preserve jira id prefix and trim only suffix.
    MAX_SUFFIX_LENGTH=$((MAX_BRANCH_LENGTH - ${#JIRA_ID_LOWER} - 1))
    if [ $MAX_SUFFIX_LENGTH -lt 1 ]; then
        MAX_SUFFIX_LENGTH=1
    fi
    
    # Truncate suffix at word boundary if possible
    TRUNCATED_SUFFIX=$(echo "$BRANCH_SUFFIX" | cut -c1-$MAX_SUFFIX_LENGTH)
    # Remove trailing hyphen if truncation created one
    TRUNCATED_SUFFIX=$(echo "$TRUNCATED_SUFFIX" | sed 's/-$//')
    
    ORIGINAL_BRANCH_NAME="$BRANCH_NAME"
    if [ -z "$TRUNCATED_SUFFIX" ]; then
        BRANCH_NAME="$JIRA_ID_LOWER"
    else
        BRANCH_NAME="${JIRA_ID_LOWER}-${TRUNCATED_SUFFIX}"
    fi
    
    >&2 echo "[specify] Warning: Branch name exceeded GitHub's 244-byte limit"
    >&2 echo "[specify] Original: $ORIGINAL_BRANCH_NAME (${#ORIGINAL_BRANCH_NAME} bytes)"
    >&2 echo "[specify] Truncated to: $BRANCH_NAME (${#BRANCH_NAME} bytes)"
fi

if [ "$HAS_GIT" = true ]; then
    git checkout -b "$BRANCH_NAME"
else
    >&2 echo "[specify] Warning: Git repository not detected; skipped branch creation for $BRANCH_NAME"
fi

FEATURE_DIR="$SPECS_DIR/$BRANCH_NAME"
mkdir -p "$FEATURE_DIR"

# Function to replace [DATE] placeholders with current date in ISO format (YYYY-MM-DD)
replace_date_placeholders() {
    local file="$1"
    local current_date=$(date +%Y-%m-%d)
    
    if [ -f "$file" ]; then
        # Use sed to replace [DATE] with current date
        if [[ "$OSTYPE" == "darwin"* ]]; then
            # macOS requires empty string for -i
            sed -i '' "s/\[DATE\]/${current_date}/g" "$file"
        else
            # Linux/other systems
            sed -i "s/\[DATE\]/${current_date}/g" "$file"
        fi
    fi
}

# Apply mode-specific defaults for options if not explicitly set
if [ -z "$TDD" ]; then
    if [ "$MODE" = "build" ]; then
        TDD="false"
    else
        TDD="false"
    fi
fi

if [ -z "$CONTRACTS" ]; then
    if [ "$MODE" = "build" ]; then
        CONTRACTS="false"
    else
        CONTRACTS="true"
    fi
fi

if [ -z "$DATA_MODELS" ]; then
    if [ "$MODE" = "build" ]; then
        DATA_MODELS="false"
    else
        DATA_MODELS="true"
    fi
fi

if [ -z "$RISK_TESTS" ]; then
    if [ "$MODE" = "build" ]; then
        RISK_TESTS="false"
    else
        RISK_TESTS="true"
    fi
fi

# Mode-aware template selection (use passed MODE, not config file)
if [ "$MODE" = "build" ]; then
    TEMPLATE="$REPO_ROOT/.cx-spec/templates/spec-template-build.md"
else
    TEMPLATE="$REPO_ROOT/.cx-spec/templates/spec-template.md"
fi
SPEC_FILE="$FEATURE_DIR/spec.md"
if [ -f "$TEMPLATE" ]; then cp "$TEMPLATE" "$SPEC_FILE"; else touch "$SPEC_FILE"; fi

# Replace [DATE] placeholders with current date
replace_date_placeholders "$SPEC_FILE"

# Replace mode and options metadata in spec.md
# Templates already have placeholders, but we need to ensure values are set correctly
if [ -f "$SPEC_FILE" ]; then
    # The templates already have the correct defaults, but if we're regenerating
    # or if template format changes, explicitly set the values
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' "s/\*\*Workflow Mode\*\*:.*/\*\*Workflow Mode\*\*: $MODE/" "$SPEC_FILE"
        sed -i '' "s/\*\*Framework Options\*\*:.*/\*\*Framework Options\*\*: tdd=$TDD, contracts=$CONTRACTS, data_models=$DATA_MODELS, risk_tests=$RISK_TESTS/" "$SPEC_FILE"
    else
        sed -i "s/\*\*Workflow Mode\*\*:.*/\*\*Workflow Mode\*\*: $MODE/" "$SPEC_FILE"
        sed -i "s/\*\*Framework Options\*\*:.*/\*\*Framework Options\*\*: tdd=$TDD, contracts=$CONTRACTS, data_models=$DATA_MODELS, risk_tests=$RISK_TESTS/" "$SPEC_FILE"
    fi
fi

CONTEXT_TEMPLATE="$REPO_ROOT/.cx-spec/templates/context-template.md"
CONTEXT_FILE="$FEATURE_DIR/context.md"

# Function to populate context.md with intelligent defaults (mode-aware)
populate_context_file() {
    local context_file="$1"
    local feature_name="$2"
    local feature_description="$3"
    local mode="$4"

    # Extract feature title (first line or first sentence)
    local feature_title=$(echo "$feature_description" | head -1 | sed 's/^[[:space:]]*//' | sed 's/[[:space:]]*$//')

    # Extract mission (first sentence, limited to reasonable length)
    local mission=$(echo "$feature_description" | grep -o '^[[:print:]]*[.!?]' | head -1 | sed 's/[.!?]$//')
    if [ -z "$mission" ]; then
        mission="$feature_description"
    fi
    # Limit mission length for readability
    if [ ${#mission} -gt 200 ]; then
        mission=$(echo "$mission" | cut -c1-200 | sed 's/[[:space:]]*$//' | sed 's/[[:space:]]*$/.../')
    fi

    # Mode-aware field population
    if [ "$mode" = "build" ]; then
        # Build mode: Minimal context, focus on core functionality
        local code_paths="To be determined during implementation"
        local directives="None (build mode)"
        local research="Minimal research needed for lightweight implementation"
    else
        # Spec mode: Comprehensive context for full specification
        # Detect code paths (basic detection based on common patterns)
        local code_paths="To be determined during planning phase"
        if echo "$feature_description" | grep -qi "api\|endpoint\|service"; then
            code_paths="api/, services/"
        elif echo "$feature_description" | grep -qi "ui\|frontend\|component"; then
            code_paths="src/components/, src/pages/"
        elif echo "$feature_description" | grep -qi "database\|data\|model"; then
            code_paths="src/models/, database/"
        fi

        # Read team directives if available
        local directives="None"
        local team_directives_root
        team_directives_root=$(get_team_directives_path)
        local team_directives_file="$team_directives_root/directives.md"
        if [ -f "$team_directives_file" ]; then
            directives="See team-ai-directives repository for applicable guidelines"
        fi

        # Set research needs
        local research="To be identified during specification and planning phases"
    fi

    # Create context.md with populated values
    cat > "$context_file" << EOF
# Feature Context

**Feature**: $feature_title
**Mission**: $mission
**Code Paths**: $code_paths
**Directives**: $directives
**Research**: $research

EOF
}

# Populate context.md with intelligent defaults
if [ -f "$CONTEXT_TEMPLATE" ]; then
    populate_context_file "$CONTEXT_FILE" "$BRANCH_SUFFIX" "$FEATURE_DESCRIPTION_NO_JIRA" "$MODE"
else
    touch "$CONTEXT_FILE"
fi

# Set the SPECIFY_FEATURE environment variable for the current session
export SPECIFY_FEATURE="$BRANCH_NAME"

if $JSON_MODE; then
    printf '{"BRANCH_NAME":"%s","SPEC_FILE":"%s","JIRA_ID":"%s"}\n' "$BRANCH_NAME" "$SPEC_FILE" "$JIRA_ID"
else
    echo "BRANCH_NAME: $BRANCH_NAME"
    echo "SPEC_FILE: $SPEC_FILE"
    echo "JIRA_ID: $JIRA_ID"
    echo "SPECIFY_FEATURE environment variable set to: $BRANCH_NAME"
fi
