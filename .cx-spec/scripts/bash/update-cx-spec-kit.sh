#!/usr/bin/env bash
# Update CX Spec Kit Script
#
# This script updates the local cx-spec-kit installation by fetching
# the latest version from the source GitHub repository.
#
# Usage: ./update-cx-spec-kit.sh
#
# Exit codes:
#   0 - Success (updated or already up to date)
#   1 - Error (network, parse, write failure)

# Only set strict mode when running directly (not when sourced for testing)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    set -e
    set -o pipefail
fi

# Configuration
GITHUB_OWNER="CheckmarxDev"
GITHUB_REPO="internal-cx-agents"
GITHUB_REF="main"
GITHUB_SUBDIR="cx-spec-kit"
GITHUB_RAW_BASE="https://raw.githubusercontent.com/${GITHUB_OWNER}/${GITHUB_REPO}/${GITHUB_REF}/${GITHUB_SUBDIR}"
GITHUB_API_BASE="https://api.github.com/repos/${GITHUB_OWNER}/${GITHUB_REPO}/contents/${GITHUB_SUBDIR}"
GITHUB_API_ROOT="repos/${GITHUB_OWNER}/${GITHUB_REPO}/contents/${GITHUB_SUBDIR}"

# Global arrays to track updated and added files
declare -a FILES_UPDATED=()
declare -a FILES_ADDED=()
declare -a FILES_FAILED=()

# Counters/state shared between update phases
LAST_UPDATED_COUNT=0
LAST_ADDED_COUNT=0
HAD_FAILURES=0

# Cached strategy/auth state
GITHUB_TOKEN_CACHE=""
GITHUB_TOKEN_RESOLVED=0
GH_AUTH_CHECKED=0
GH_AUTH_READY=0
FETCH_STRATEGY_ANNOUNCED=0

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Output functions
print_error() { echo -e "${RED}❌ ERROR: $1${NC}" >&2; }
print_success() { echo -e "${GREEN}✅ $1${NC}"; }
print_warning() { echo -e "${YELLOW}⚠️  $1${NC}"; }
print_info() { echo -e "${BLUE}ℹ️  $1${NC}"; }
print_step() { echo -e "${GREEN}🔄 $1${NC}"; }

# Return success when value contains non-whitespace characters
has_non_whitespace() {
    local value="$1"
    [[ -n "${value//[[:space:]]/}" ]]
}

# Reset cached auth/fetch state
reset_github_token_cache() {
    GITHUB_TOKEN_CACHE=""
    GITHUB_TOKEN_RESOLVED=0
    GH_AUTH_CHECKED=0
    GH_AUTH_READY=0
}

set_last_counts() {
    LAST_UPDATED_COUNT="$1"
    LAST_ADDED_COUNT="$2"
}

record_failure() {
    local path="$1"
    HAD_FAILURES=1
    FILES_FAILED+=("$path")
}

# Decode base64 in a cross-platform way
decode_base64() {
    if base64 --decode >/dev/null 2>&1 <<< ""; then
        base64 --decode
    elif base64 -D >/dev/null 2>&1 <<< ""; then
        base64 -D
    else
        base64 -d
    fi
}

# Parse GitHub contents API payload and emit "<type>\t<name>" rows
parse_github_entries() {
    if command -v jq >/dev/null 2>&1; then
        jq -r '.[] | select(.type == "file" or .type == "dir") | [.type, .name] | @tsv'
        return
    fi

    # Fallback parser when jq is unavailable.
    tr -d '\n' \
        | sed -e 's/^\[//' -e 's/\]$//' -e 's/},{/}\n{/g' \
        | while IFS= read -r entry; do
            local entry_type=""
            local entry_name=""

            if [[ "$entry" =~ \"type\"[[:space:]]*:[[:space:]]*\"([^\"]+)\" ]]; then
                entry_type="${BASH_REMATCH[1]}"
            fi
            if [[ "$entry" =~ \"name\"[[:space:]]*:[[:space:]]*\"([^\"]+)\" ]]; then
                entry_name="${BASH_REMATCH[1]}"
            fi

            if [[ -n "$entry_type" && -n "$entry_name" && ( "$entry_type" == "file" || "$entry_type" == "dir" ) ]]; then
                printf '%s\t%s\n' "$entry_type" "$entry_name"
            fi
        done
}

# Resolve GitHub token from env only (strategy 2)
get_github_token() {
    if [[ "$GITHUB_TOKEN_RESOLVED" -eq 1 ]]; then
        echo "$GITHUB_TOKEN_CACHE"
        return
    fi

    local token=""

    if [[ -n "${GH_TOKEN:-}" ]]; then
        token="$GH_TOKEN"
    elif [[ -n "${GITHUB_TOKEN:-}" ]]; then
        token="$GITHUB_TOKEN"
    fi

    GITHUB_TOKEN_CACHE="$token"
    GITHUB_TOKEN_RESOLVED=1
    echo "$token"
}

# Check whether GitHub CLI auth is available (strategy 1)
gh_cli_auth_ready() {
    if [[ "$GH_AUTH_CHECKED" -eq 1 ]]; then
        [[ "$GH_AUTH_READY" -eq 1 ]]
        return
    fi

    GH_AUTH_CHECKED=1
    if command -v gh >/dev/null 2>&1 && gh auth status -h github.com >/dev/null 2>&1; then
        GH_AUTH_READY=1
    else
        GH_AUTH_READY=0
    fi

    [[ "$GH_AUTH_READY" -eq 1 ]]
}

# Keep for backward compatibility with tests/callers.
# Update flow no longer requires preflight auth; fetch fallback handles it.
ensure_github_auth() {
    return 0
}

# Announce chosen fetch strategy once
announce_fetch_strategy() {
    local strategy="$1"
    [[ "$FETCH_STRATEGY_ANNOUNCED" -eq 1 ]] && return
    print_info "Using GitHub fetch strategy: $strategy"
    FETCH_STRATEGY_ANNOUNCED=1
}

# Manual fallback guidance (strategy 4)
print_manual_fetch_help() {
    local path="$1"
    print_error "Unable to fetch '$path' after trying all automated strategies:"
    print_error "1) GitHub CLI (gh auth)"
    print_error "2) GH_TOKEN/GITHUB_TOKEN"
    print_error "3) Public web fetch"
    print_error "Manual fallback: https://github.com/${GITHUB_OWNER}/${GITHUB_REPO}/blob/${GITHUB_REF}/${GITHUB_SUBDIR}/${path}"
    print_error "Or run: gh api repos/${GITHUB_OWNER}/${GITHUB_REPO}/contents/${GITHUB_SUBDIR}/${path} --jq '.content' | base64 -d"
}

# Strategy 1: GitHub CLI
fetch_via_gh_cli_content() {
    local remote_path="$1"
    local encoded=""
    gh_cli_auth_ready || return 1
    encoded=$(gh api "${GITHUB_API_ROOT}/${remote_path}?ref=${GITHUB_REF}" --jq '.content' 2>/dev/null | tr -d '\n\r') || return 1
    has_non_whitespace "$encoded" || return 1
    printf '%s' "$encoded" | decode_base64
}

fetch_via_gh_cli_file_list() {
    local path="$1"
    local files=""
    gh_cli_auth_ready || return 1
    files=$(gh api "${GITHUB_API_ROOT}/${path}?ref=${GITHUB_REF}" --jq '.[] | select(.type == "file") | .name' 2>/dev/null) || return 1
    has_non_whitespace "$files" || return 1
    printf '%s\n' "$files"
}

fetch_via_gh_cli_entries() {
    local path="$1"
    local entries=""
    gh_cli_auth_ready || return 1
    entries=$(gh api "${GITHUB_API_ROOT}/${path}?ref=${GITHUB_REF}" --jq '.[] | select(.type == "file" or .type == "dir") | "\(.type)\t\(.name)"' 2>/dev/null) || return 1
    has_non_whitespace "$entries" || return 1
    printf '%s\n' "$entries"
}

fetch_via_gh_cli_to_file() {
    local remote_path="$1"
    local output_file="$2"
    local encoded=""
    gh_cli_auth_ready || return 1
    encoded=$(gh api "${GITHUB_API_ROOT}/${remote_path}?ref=${GITHUB_REF}" --jq '.content' 2>/dev/null | tr -d '\n\r') || return 1
    has_non_whitespace "$encoded" || return 1
    printf '%s' "$encoded" | decode_base64 > "$output_file"
    [[ -s "$output_file" ]]
}

# Strategy 2: GitHub API/raw with env token
fetch_via_token_content() {
    local remote_path="$1"
    local token
    token="$(get_github_token)"
    [[ -n "$token" ]] || return 1
    curl -fsSL -H "Authorization: Bearer $token" "${GITHUB_RAW_BASE}/${remote_path}"
}

fetch_via_token_file_list() {
    local path="$1"
    local token
    local response=""
    local files=""
    token="$(get_github_token)"
    [[ -n "$token" ]] || return 1
    response=$(curl -fsSL \
        -H "Authorization: Bearer $token" \
        -H "Accept: application/vnd.github+json" \
        "${GITHUB_API_BASE}/${path}?ref=${GITHUB_REF}") || return 1
    files=$(printf '%s' "$response" \
        | grep -o '"name"[[:space:]]*:[[:space:]]*"[^"]*"' \
        | sed 's/.*"name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/') || true
    has_non_whitespace "$files" || return 1
    printf '%s\n' "$files"
}

fetch_via_token_entries() {
    local path="$1"
    local token
    local response=""
    local entries=""
    token="$(get_github_token)"
    [[ -n "$token" ]] || return 1
    response=$(curl -fsSL \
        -H "Authorization: Bearer $token" \
        -H "Accept: application/vnd.github+json" \
        "${GITHUB_API_BASE}/${path}?ref=${GITHUB_REF}") || return 1
    entries=$(printf '%s' "$response" | parse_github_entries) || return 1
    has_non_whitespace "$entries" || return 1
    printf '%s\n' "$entries"
}

fetch_via_token_to_file() {
    local remote_path="$1"
    local output_file="$2"
    local token
    token="$(get_github_token)"
    [[ -n "$token" ]] || return 1
    curl -fsSL -H "Authorization: Bearer $token" "${GITHUB_RAW_BASE}/${remote_path}" -o "$output_file"
}

# Strategy 3: Public web fetch
fetch_via_web_content() {
    local remote_path="$1"
    curl -fsSL "${GITHUB_RAW_BASE}/${remote_path}"
}

fetch_via_web_file_list() {
    local path="$1"
    local response=""
    local files=""
    response=$(curl -fsSL "${GITHUB_API_BASE}/${path}?ref=${GITHUB_REF}") || return 1
    files=$(printf '%s' "$response" \
        | grep -o '"name"[[:space:]]*:[[:space:]]*"[^"]*"' \
        | sed 's/.*"name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/') || true
    has_non_whitespace "$files" || return 1
    printf '%s\n' "$files"
}

fetch_via_web_entries() {
    local path="$1"
    local response=""
    local entries=""
    response=$(curl -fsSL "${GITHUB_API_BASE}/${path}?ref=${GITHUB_REF}") || return 1
    entries=$(printf '%s' "$response" | parse_github_entries) || return 1
    has_non_whitespace "$entries" || return 1
    printf '%s\n' "$entries"
}

fetch_via_web_to_file() {
    local remote_path="$1"
    local output_file="$2"
    curl -fsSL "${GITHUB_RAW_BASE}/${remote_path}" -o "$output_file"
}

# Strategy orchestrators
fetch_remote_content() {
    local remote_path="$1"

    local response=""
    if response=$(fetch_via_gh_cli_content "$remote_path" 2>/dev/null); then
        if has_non_whitespace "$response"; then
            announce_fetch_strategy "GitHub CLI (gh auth)"
            echo "$response"
            return 0
        fi
    fi

    if response=$(fetch_via_token_content "$remote_path" 2>/dev/null); then
        if has_non_whitespace "$response"; then
            announce_fetch_strategy "GH_TOKEN/GITHUB_TOKEN"
            echo "$response"
            return 0
        fi
    fi

    if response=$(fetch_via_web_content "$remote_path" 2>/dev/null); then
        if has_non_whitespace "$response"; then
            announce_fetch_strategy "Public web fetch"
            echo "$response"
            return 0
        fi
    fi

    print_manual_fetch_help "$remote_path"
    return 1
}

fetch_remote_file_list() {
    local path="$1"
    local files=""

    if files=$(fetch_via_gh_cli_file_list "$path" 2>/dev/null); then
        if has_non_whitespace "$files"; then
            announce_fetch_strategy "GitHub CLI (gh auth)"
            echo "$files"
            return 0
        fi
    fi

    if files=$(fetch_via_token_file_list "$path" 2>/dev/null); then
        if has_non_whitespace "$files"; then
            announce_fetch_strategy "GH_TOKEN/GITHUB_TOKEN"
            echo "$files"
            return 0
        fi
    fi

    if files=$(fetch_via_web_file_list "$path" 2>/dev/null); then
        if has_non_whitespace "$files"; then
            announce_fetch_strategy "Public web fetch"
            echo "$files"
            return 0
        fi
    fi

    print_manual_fetch_help "$path"
    return 1
}

fetch_remote_entries() {
    local path="$1"
    local entries=""

    if entries=$(fetch_via_gh_cli_entries "$path" 2>/dev/null); then
        if has_non_whitespace "$entries"; then
            announce_fetch_strategy "GitHub CLI (gh auth)"
            echo "$entries"
            return 0
        fi
    fi

    if entries=$(fetch_via_token_entries "$path" 2>/dev/null); then
        if has_non_whitespace "$entries"; then
            announce_fetch_strategy "GH_TOKEN/GITHUB_TOKEN"
            echo "$entries"
            return 0
        fi
    fi

    if entries=$(fetch_via_web_entries "$path" 2>/dev/null); then
        if has_non_whitespace "$entries"; then
            announce_fetch_strategy "Public web fetch"
            echo "$entries"
            return 0
        fi
    fi

    print_manual_fetch_help "$path"
    return 1
}

fetch_file_list_recursive() {
    local path="$1"
    local prefix="$2"
    local entries
    entries=$(fetch_remote_entries "$path") || return 1

    local entry_type=""
    local entry_name=""
    while IFS=$'\t' read -r entry_type entry_name; do
        [[ -z "$entry_type" || -z "$entry_name" ]] && continue

        if [[ "$entry_type" == "file" ]]; then
            if [[ -n "$prefix" ]]; then
                printf '%s/%s\n' "$prefix" "$entry_name"
            else
                printf '%s\n' "$entry_name"
            fi
            continue
        fi

        if [[ "$entry_type" == "dir" ]]; then
            local child_path="$path/$entry_name"
            local child_prefix="$entry_name"
            [[ -n "$prefix" ]] && child_prefix="$prefix/$entry_name"

            local nested_files
            nested_files=$(fetch_file_list_recursive "$child_path" "$child_prefix") || return 1
            if [[ -n "$nested_files" ]]; then
                printf '%s\n' "$nested_files"
            fi
        fi
    done <<< "$entries"
}

download_remote_to_file() {
    local remote_path="$1"
    local output_file="$2"

    if fetch_via_gh_cli_to_file "$remote_path" "$output_file" 2>/dev/null; then
        if [[ -s "$output_file" ]]; then
            announce_fetch_strategy "GitHub CLI (gh auth)"
            return 0
        fi
    fi

    if fetch_via_token_to_file "$remote_path" "$output_file" 2>/dev/null; then
        if [[ -s "$output_file" ]]; then
            announce_fetch_strategy "GH_TOKEN/GITHUB_TOKEN"
            return 0
        fi
    fi

    if fetch_via_web_to_file "$remote_path" "$output_file" 2>/dev/null; then
        if [[ -s "$output_file" ]]; then
            announce_fetch_strategy "Public web fetch"
            return 0
        fi
    fi

    print_manual_fetch_help "$remote_path"
    return 1
}

# Get repository root
get_repo_root() {
    if git rev-parse --show-toplevel >/dev/null 2>&1; then
        git rev-parse --show-toplevel
    else
        # Fall back to script location
        local script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
        echo "$(cd "$script_dir/../../.." && pwd)"
    fi
}

# Compare semantic versions
# Returns: 0 if v1 > v2, 1 if v1 = v2, 2 if v1 < v2
compare_versions() {
    local v1="$1"
    local v2="$2"

    # Handle empty versions
    [[ -z "$v1" ]] && v1="0"
    [[ -z "$v2" ]] && v2="0"

    # Split versions into arrays
    IFS='.' read -ra V1_PARTS <<< "$v1"
    IFS='.' read -ra V2_PARTS <<< "$v2"

    # Compare each part
    local max_parts=${#V1_PARTS[@]}
    [[ ${#V2_PARTS[@]} -gt $max_parts ]] && max_parts=${#V2_PARTS[@]}

    for ((i=0; i<max_parts; i++)); do
        local p1="${V1_PARTS[i]:-0}"
        local p2="${V2_PARTS[i]:-0}"

        if [[ $p1 -gt $p2 ]]; then
            echo "0"  # v1 > v2
            return
        elif [[ $p1 -lt $p2 ]]; then
            echo "2"  # v1 < v2
            return
        fi
    done

    echo "1"  # v1 = v2
}

# Fetch remote config and extract version
fetch_remote_version() {
    local response

    if ! response=$(fetch_remote_content "config.json"); then
        print_error "Failed to fetch remote config"
        return 1
    fi

    # Extract version using grep/sed (no jq dependency)
    local version
    version=$(echo "$response" | grep -o '"version"[[:space:]]*:[[:space:]]*"[^"]*"' | head -1 | sed 's/.*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')

    if [[ -z "$version" ]]; then
        print_error "Could not parse version from remote config"
        return 1
    fi

    echo "$version"
}

# Get local version from config
get_local_version() {
    local repo_root="$1"
    local config_file="$repo_root/.cx-spec/config.json"

    if [[ ! -f "$config_file" ]]; then
        echo ""
        return
    fi

    # Extract version using grep/sed
    local version
    version=$(grep -o '"version"[[:space:]]*:[[:space:]]*"[^"]*"' "$config_file" 2>/dev/null | head -1 | sed 's/.*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')

    echo "$version"
}

# Fetch file list from GitHub API for a directory
fetch_file_list() {
    local path="$1"
    fetch_file_list_recursive "$path" ""
}

# Download and save a file
download_file() {
    local remote_path="$1"
    local local_path="$2"
    local parent_dir
    parent_dir=$(dirname "$local_path")

    # Create parent directory if needed
    if [[ ! -d "$parent_dir" ]]; then
        mkdir -p "$parent_dir"
    fi

    if ! download_remote_to_file "$remote_path" "$local_path"; then
        return 1
    fi

    return 0
}


# Merge configs: preserve user settings, update version and add new keys
merge_configs() {
    local repo_root="$1"
    local remote_config="$2"
    local local_config_file="$repo_root/.cx-spec/config.json"

    if [[ ! -f "$local_config_file" ]]; then
        # No local config, just use remote
        if ! printf '%s\n' "$remote_config" > "$local_config_file"; then
            return 1
        fi
        return 0
    fi

    local local_config
    if ! local_config=$(cat "$local_config_file"); then
        return 1
    fi

    # If jq is available, use it for proper merging
    if command -v jq >/dev/null 2>&1; then
        # Merge: remote as base, preserve user-customized fields from local
        local merged
        if ! merged=$(jq -s '
            .[0] as $remote | .[1] as $local |
            $remote * {
                workflow: ($local.workflow // $remote.workflow),
                options: ($local.options // $remote.options),
                mode_defaults: ($local.mode_defaults // $remote.mode_defaults),
                spec_sync: ($local.spec_sync // $remote.spec_sync),
                team_directives: ($local.team_directives // $remote.team_directives),
                architecture: ($local.architecture // $remote.architecture)
            } | .version = $remote.version
        ' <(echo "$remote_config") <(echo "$local_config")); then
            return 1
        fi

        if ! printf '%s\n' "$merged" > "$local_config_file"; then
            return 1
        fi
    else
        # Fallback: just update version in local config
        local remote_version
        remote_version=$(echo "$remote_config" | grep -o '"version"[[:space:]]*:[[:space:]]*"[^"]*"' | head -1)
        if [[ -z "$remote_version" ]]; then
            return 1
        fi

        # Replace version line in local config
        if ! sed -i.bak "s/\"version\"[[:space:]]*:[[:space:]]*\"[^\"]*\"/$remote_version/" "$local_config_file"; then
            rm -f "${local_config_file}.bak"
            return 1
        fi
        rm -f "${local_config_file}.bak"
    fi

    return 0
}

# Extract command description from frontmatter
extract_command_description() {
    local source_file="$1"

    awk '
        NR == 1 && $0 == "---" { in_frontmatter = 1; next }
        in_frontmatter && $0 == "---" { exit }
        in_frontmatter && $0 ~ /^description:[[:space:]]*/ {
            sub(/^description:[[:space:]]*/, "", $0)
            gsub(/^["'"'"']|["'"'"']$/, "", $0)
            print
            exit
        }
    ' "$source_file"
}

build_codex_skill_from_command() {
    local source_file="$1"
    local target_file="$2"
    local skill_name="$3"

    local description
    description=$(extract_command_description "$source_file")
    [[ -z "$description" ]] && description="CX Spec Kit workflow skill"

    mkdir -p "$(dirname "$target_file")"

    {
        echo "---"
        echo "name: $skill_name"
        echo "description: >"
        echo "  $description"
        echo "---"
        echo ""
        echo "## Argument Handling"
        echo ""
        echo "- Invocation format: \`$skill_name <arguments>\`"
        echo "- Treat text after the skill name as \`\$ARGUMENTS\`."
        echo "- If no arguments are provided, follow the empty-input behavior already documented below."
        echo ""
        awk '
            NR == 1 && $0 == "---" { in_frontmatter = 1; next }
            in_frontmatter && $0 == "---" { in_frontmatter = 0; next }
            !in_frontmatter { print }
        ' "$source_file"
    } > "$target_file"
}

# Update command assets for a specific AI target
update_commands() {
    local repo_root="$1"
    local target="$2"
    local target_dir
    local target_display
    local updated=0
    local added=0

    if [[ "$target" == "codex" ]]; then
        target_dir="$repo_root/.codex/skills"
        target_display=".codex/skills"
        print_step "Updating codex skills..." >&2
    else
        target_dir="$repo_root/.${target}/commands"
        target_display=".$target/commands"
        print_step "Updating $target commands..." >&2
    fi

    # Get list of command files from GitHub
    local files
    if ! files=$(fetch_file_list "commands"); then
        print_warning "Failed to fetch command file list" >&2
        record_failure "$target_display/*"
        set_last_counts 0 0
        echo "0 0"
        return
    fi

    local file
    while IFS= read -r file; do
        [[ -z "$file" ]] && continue
        [[ "$file" != *.md ]] && continue

        local local_file
        local display_path

        if [[ "$target" == "codex" ]]; then
            local skill_name="${file%.md}"
            skill_name="${skill_name//\//.}"
            local_file="$target_dir/$skill_name/SKILL.md"
            display_path=".codex/skills/$skill_name/SKILL.md"
        else
            local_file="$target_dir/$file"
            display_path=".$target/commands/$file"
        fi

        local existed=0
        [[ -f "$local_file" ]] && existed=1

        if [[ "$target" == "codex" ]]; then
            local temp_command_file
            temp_command_file=$(mktemp)

            if ! download_file "commands/$file" "$temp_command_file"; then
                rm -f "$temp_command_file"
                print_warning "Failed to download $file" >&2
                record_failure "$display_path"
                continue
            fi

            if ! build_codex_skill_from_command "$temp_command_file" "$local_file" "$skill_name"; then
                rm -f "$temp_command_file"
                print_warning "Failed to generate codex skill for $file" >&2
                record_failure "$display_path"
                continue
            fi

            rm -f "$temp_command_file"
        else
            if ! download_file "commands/$file" "$local_file"; then
                print_warning "Failed to download $file" >&2
                record_failure "$display_path"
                continue
            fi
        fi

        if [[ "$existed" -eq 1 ]]; then
            updated=$((updated + 1))
            FILES_UPDATED+=("$display_path")
        else
            added=$((added + 1))
            FILES_ADDED+=("$display_path")
        fi
    done <<< "$files"

    set_last_counts "$updated" "$added"
    echo "$updated $added"
}

# Update scripts (bash and powershell)
update_scripts() {
    local repo_root="$1"
    local updated=0
    local added=0

    print_step "Updating scripts..." >&2

    # Update bash scripts
    local bash_files
    if ! bash_files=$(fetch_file_list "scripts/bash"); then
        print_warning "Failed to fetch bash script file list" >&2
        record_failure ".cx-spec/scripts/bash/*"
        set_last_counts 0 0
        echo "0 0"
        return
    fi

    local file
    while IFS= read -r file; do
        [[ -z "$file" ]] && continue
        [[ "$file" != *.sh ]] && continue

        local local_file="$repo_root/.cx-spec/scripts/bash/$file"
        local display_path=".cx-spec/scripts/bash/$file"
        local existed=0
        [[ -f "$local_file" ]] && existed=1

        if download_file "scripts/bash/$file" "$local_file"; then
            chmod +x "$local_file" || true
            if [[ "$existed" -eq 1 ]]; then
                updated=$((updated + 1))
                FILES_UPDATED+=("$display_path")
            else
                added=$((added + 1))
                FILES_ADDED+=("$display_path")
            fi
        else
            print_warning "Failed to download bash/$file" >&2
            record_failure "$display_path"
        fi
    done <<< "$bash_files"

    # Update powershell scripts
    local ps_files
    if ! ps_files=$(fetch_file_list "scripts/powershell"); then
        print_warning "Failed to fetch PowerShell script file list" >&2
        record_failure ".cx-spec/scripts/powershell/*"
        set_last_counts "$updated" "$added"
        echo "$updated $added"
        return
    fi

    while IFS= read -r file; do
        [[ -z "$file" ]] && continue
        [[ "$file" != *.ps1 ]] && continue

        local local_file="$repo_root/.cx-spec/scripts/powershell/$file"
        local display_path=".cx-spec/scripts/powershell/$file"
        local existed=0
        [[ -f "$local_file" ]] && existed=1

        if ! download_file "scripts/powershell/$file" "$local_file"; then
            print_warning "Failed to download powershell/$file" >&2
            record_failure "$display_path"
            continue
        fi

        if [[ "$existed" -eq 1 ]]; then
            updated=$((updated + 1))
            FILES_UPDATED+=("$display_path")
        else
            added=$((added + 1))
            FILES_ADDED+=("$display_path")
        fi
    done <<< "$ps_files"

    set_last_counts "$updated" "$added"
    echo "$updated $added"
}

# Detect which AI target directories exist
detect_ai_targets() {
    local repo_root="$1"
    local targets=""

    [[ -d "$repo_root/.cursor/commands" ]] && targets="$targets cursor"
    [[ -d "$repo_root/.claude/commands" ]] && targets="$targets claude"
    [[ -d "$repo_root/.codex/skills" ]] && targets="$targets codex"

    echo "$targets"
}

# Update templates
update_templates() {
    local repo_root="$1"
    local updated=0
    local added=0

    print_step "Updating templates..." >&2

    local files
    if ! files=$(fetch_file_list "templates"); then
        print_warning "Failed to fetch template file list" >&2
        record_failure ".cx-spec/templates/*"
        set_last_counts 0 0
        echo "0 0"
        return
    fi

    local file
    while IFS= read -r file; do
        [[ -z "$file" ]] && continue
        [[ "$file" != *.md ]] && continue

        local local_file="$repo_root/.cx-spec/templates/$file"
        local display_path=".cx-spec/templates/$file"
        local existed=0
        [[ -f "$local_file" ]] && existed=1

        if ! download_file "templates/$file" "$local_file"; then
            print_warning "Failed to download template $file" >&2
            record_failure "$display_path"
            continue
        fi

        if [[ "$existed" -eq 1 ]]; then
            updated=$((updated + 1))
            FILES_UPDATED+=("$display_path")
        else
            added=$((added + 1))
            FILES_ADDED+=("$display_path")
        fi
    done <<< "$files"

    set_last_counts "$updated" "$added"
    echo "$updated $added"
}

# Convert array to JSON array string
array_to_json() {
    if [[ $# -eq 0 ]]; then
        echo "[]"
        return
    fi

    printf '%s\0' "$@" | jq -Rs 'split("\u0000")[:-1]'
}

# Output JSON result
output_result() {
    local status="$1"
    local local_version="$2"
    local remote_version="$3"
    local files_updated="$4"
    local files_added="$5"
    local files_failed="$6"
    local message="$7"

    if ! command -v jq >/dev/null 2>&1; then
        print_error "jq is required to format update output JSON"
        return 1
    fi

    local updated_json
    local added_json
    local failed_json
    updated_json=$(array_to_json "${FILES_UPDATED[@]}")
    added_json=$(array_to_json "${FILES_ADDED[@]}")
    failed_json=$(array_to_json "${FILES_FAILED[@]}")

    jq -n \
        --arg status "$status" \
        --arg local_version "$local_version" \
        --arg remote_version "$remote_version" \
        --arg message "$message" \
        --argjson files_updated_count "${files_updated:-0}" \
        --argjson files_added_count "${files_added:-0}" \
        --argjson files_failed_count "${files_failed:-0}" \
        --argjson files_updated "$updated_json" \
        --argjson files_added "$added_json" \
        --argjson files_failed "$failed_json" \
        '{
            status: $status,
            local_version: $local_version,
            remote_version: $remote_version,
            files_updated_count: $files_updated_count,
            files_added_count: $files_added_count,
            files_failed_count: $files_failed_count,
            files_updated: $files_updated,
            files_added: $files_added,
            files_failed: $files_failed,
            message: $message
        }'
}

emit_jq_missing_error() {
    print_error "jq is required to run /cx-spec.update. Install jq and retry."
    printf '%s\n' '{"status":"error","local_version":"","remote_version":"","files_updated_count":0,"files_added_count":0,"files_failed_count":0,"files_updated":[],"files_added":[],"files_failed":[],"message":"jq is required to run /cx-spec.update"}'
}

# Main function
main() {
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "🔄 CX Spec Kit Update"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""

    FILES_UPDATED=()
    FILES_ADDED=()
    FILES_FAILED=()
    set_last_counts 0 0
    HAD_FAILURES=0
    FETCH_STRATEGY_ANNOUNCED=0
    reset_github_token_cache

    if ! command -v jq >/dev/null 2>&1; then
        emit_jq_missing_error
        exit 1
    fi

    local repo_root
    repo_root=$(get_repo_root)

    print_info "Repository root: $repo_root"

    # Step 1: Fetch remote version
    print_step "Checking for updates..."

    local remote_version
    if ! remote_version=$(fetch_remote_version); then
        print_error "Failed to check for updates"
        output_result "error" "" "" 0 0 0 "Failed to fetch remote version"
        exit 1
    fi

    # Step 2: Get local version
    local local_version
    local_version=$(get_local_version "$repo_root")

    print_info "Local version: ${local_version:-"(not installed)"}"
    print_info "Remote version: $remote_version"

    # Step 3: Compare versions
    if [[ -n "$local_version" ]]; then
        local cmp_result
        cmp_result=$(compare_versions "$remote_version" "$local_version")

        if [[ "$cmp_result" != "0" ]]; then
            print_success "Already up to date (v$local_version)"
            echo ""
            output_result "up_to_date" "$local_version" "$remote_version" 0 0 0 "Already up to date"
            exit 0
        fi
    fi

    # Step 4: Perform update
    echo ""
    print_info "Updating from ${local_version:-"(none)"} to $remote_version..."
    echo ""

    local total_updated=0
    local total_added=0

    # Update scripts
    update_scripts "$repo_root" >/dev/null
    total_updated=$((total_updated + LAST_UPDATED_COUNT))
    total_added=$((total_added + LAST_ADDED_COUNT))

    # Update templates
    update_templates "$repo_root" >/dev/null
    total_updated=$((total_updated + LAST_UPDATED_COUNT))
    total_added=$((total_added + LAST_ADDED_COUNT))

    # Update commands for each detected AI target
    local ai_targets
    ai_targets=$(detect_ai_targets "$repo_root")

    for target in $ai_targets; do
        update_commands "$repo_root" "$target" >/dev/null
        total_updated=$((total_updated + LAST_UPDATED_COUNT))
        total_added=$((total_added + LAST_ADDED_COUNT))
    done

    # Update config (merge)
    if [[ "$HAD_FAILURES" -eq 0 ]]; then
        print_step "Updating configuration..."
        local remote_config
        if remote_config=$(fetch_remote_content "config.json" 2>/dev/null); then
            if ! merge_configs "$repo_root" "$remote_config"; then
                print_warning "Failed to merge configuration" >&2
                record_failure ".cx-spec/config.json"
            fi
        else
            print_warning "Failed to update configuration"
            record_failure ".cx-spec/config.json"
        fi
    else
        print_warning "Skipping configuration update because earlier update steps failed"
    fi

    local total_failed=${#FILES_FAILED[@]}

    # Summary
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    if [[ "$HAD_FAILURES" -eq 1 ]]; then
        print_error "Update finished with errors"
    else
        print_success "Update complete!"
    fi
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    print_info "Version: ${local_version:-"(none)"} → $remote_version"
    echo ""

    # Print updated files
    if [[ ${#FILES_UPDATED[@]} -gt 0 ]]; then
        echo -e "${BLUE}📝 Files updated (${#FILES_UPDATED[@]}):${NC}"
        for file in "${FILES_UPDATED[@]}"; do
            echo "   • $file"
        done
        echo ""
    fi

    # Print added files
    if [[ ${#FILES_ADDED[@]} -gt 0 ]]; then
        echo -e "${GREEN}✨ Files added (${#FILES_ADDED[@]}):${NC}"
        for file in "${FILES_ADDED[@]}"; do
            echo "   • $file"
        done
        echo ""
    fi

    # Print failed files
    if [[ ${#FILES_FAILED[@]} -gt 0 ]]; then
        echo -e "${RED}❌ Files failed (${#FILES_FAILED[@]}):${NC}"
        for file in "${FILES_FAILED[@]}"; do
            echo "   • $file"
        done
        echo ""
    fi

    if [[ "$HAD_FAILURES" -eq 1 ]]; then
        output_result "error" "${local_version:-""}" "$remote_version" "$total_updated" "$total_added" "$total_failed" "Update failed before completion; local version was kept at ${local_version:-"(none)"}."
        exit 1
    fi

    output_result "updated" "${local_version:-""}" "$remote_version" "$total_updated" "$total_added" "$total_failed" "Updated from ${local_version:-"(none)"} to $remote_version"
}

# Run main function only if executed directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
