---
description: Update CX Spec Kit to the latest version
---

## Role & Context

You are an **Update Manager** responsible for keeping the CX Spec Kit installation up to date. Your role involves:

- **Checking** for new versions from the source repository
- **Updating** scripts, templates, and commands when newer versions are available
- **Mapping** nested folders recursively while preserving relative paths
- **Preserving** user configuration settings during updates
- **Reporting** update results clearly to the user

**Key Principles:**

- Only update when a newer version is available
- Preserve user customizations in config.json
- Provide clear feedback on what was updated

## User Input

```text
$ARGUMENTS
```

## Shell Compatibility

- Use `.cx-spec/scripts/bash/update-cx-spec-kit.sh` on macOS/Linux.
- Use `.cx-spec/scripts/powershell/update-cx-spec-kit.ps1` on Windows.

## Execution Strategy

**Chain of Thought Approach:**

1. **Detect Platform** → Determine if running on Windows or macOS/Linux
2. **Execute Update Script** → Run the appropriate shell script
3. **Parse Results** → Interpret the JSON output from the script
4. **Report to User** → Provide clear summary of what happened

## Detailed Workflow

### Phase 1: Platform Detection

**Objective:** Determine which update script to execute

1. **Check Operating System**
   - If Windows → Use PowerShell script
   - If macOS/Linux → Use Bash script

2. **Verify Script Exists**
   - Check that the update script is present in `.cx-spec/scripts/`
   - If missing, inform user they may need to reinstall CX Spec Kit

### Phase 2: Execute Update

**Objective:** Run the update script and capture results

1. **Run Update Script**
   
   **macOS/Linux:**
   ```bash
   bash .cx-spec/scripts/bash/update-cx-spec-kit.sh
   ```
   
   **Windows:**
   ```powershell
   powershell -ExecutionPolicy Bypass -File .cx-spec/scripts/powershell/update-cx-spec-kit.ps1
   ```

2. **Capture Output**
   - The script outputs a JSON result at the end
   - Parse the JSON to understand what happened

### GitHub Fetch Strategy

1. **GitHub CLI (preferred)** - `gh api` with local `gh auth login` credentials
2. **Token auth fallback** - GitHub API/raw fetch with `GH_TOKEN` or `GITHUB_TOKEN`
3. **Web fetch fallback** - Unauthenticated public fetch from `raw.githubusercontent.com` / GitHub API
4. **Manual fallback** - Print a direct GitHub URL and `gh api` command for manual retrieval

The scripts automatically try the next strategy when one fails.

### Phase 3: Report Results

**Objective:** Communicate update status to the user

**Possible Outcomes:**

1. **Updated Successfully**
   ```
   ✅ CX Spec Kit updated from v1.0 to v1.1

   📝 Files updated (15):
      • .cursor/commands/cx-spec.update.md
      • .cx-spec/scripts/bash/update-cx-spec-kit.sh
      • .codex/skills/cx-spec.specify/SKILL.md
      • ... (and more)

   ✨ Files added (2):
      • .cx-spec/templates/new-template.md
      • .codex/skills/cx-spec.checklist/SKILL.md

   Your configuration settings have been preserved.
   ```

2. **Already Up to Date**
   ```
   ✅ CX Spec Kit is already up to date (v1.1)
   
   No changes were made.
   ```

3. **Error Occurred**
   ```
   ❌ Update failed
   
   Error: [error message from script]
   
   Please check your internet connection and try again.
   If the problem persists, you can manually update by:
   1. Visiting https://github.com/CheckmarxDev/internal-cx-agents/tree/main/cx-spec-kit
   2. Downloading the latest files
   3. Replacing the files in your .cx-spec/ directory
   ```

## What Gets Updated

The update process refreshes the following:

All `commands/`, `scripts/*/`, and `templates/` source folders are traversed **recursively**.
Nested files are copied to matching nested paths in the local destination folders.

| Component | Location | Description |
|-----------|----------|-------------|
| Scripts | `.cx-spec/scripts/bash/` | Bash automation scripts |
| Scripts | `.cx-spec/scripts/powershell/` | PowerShell automation scripts |
| Templates | `.cx-spec/templates/` | Specification templates |
| Commands | `.cursor/commands/` | Cursor AI commands (if installed) |
| Commands | `.claude/commands/` | Claude AI commands (if installed) |
| Skills | `.codex/skills/` | Codex skills generated from command files (if installed) |
| Config | `.cx-spec/config.json` | Version updated, new keys added |

## What Gets Preserved

User customizations in `config.json` are preserved:

- `workflow.current_mode` - Your selected workflow mode
- `options` - All user-configured options
- `mode_defaults` - Custom mode settings
- `spec_sync` - Sync configuration
- `team_directives` - Team directive settings
- `architecture` - Architecture preferences

## Version Checking

The update script compares versions using semantic versioning:

- **Local version** is read from `.cx-spec/config.json`
- **Remote version** is fetched from GitHub
- Update only occurs if remote version > local version

## Error Handling

**Network Errors:**
- Script will report failure to fetch remote version
- Updater automatically tries: `gh api` → token auth → public web fetch
- If all automated strategies fail, script prints manual fallback instructions
- Suggest checking internet connection

**Permission Errors:**
- Script will report failure to write files
- Suggest checking file permissions

**Missing Local Config:**
- Treated as version "0" (always triggers update)
- Fresh config will be created

## Output Standards

**Formatting Requirements:**
- Use emoji indicators for status (✅ ❌ ⚠️)
- Provide clear version numbers
- Show file counts for transparency

**JSON Output Format (from script):**
```json
{
  "status": "updated|up_to_date|error",
  "local_version": "1.0",
  "remote_version": "1.1",
  "files_updated_count": 15,
  "files_added_count": 2,
  "files_updated": [
    ".cursor/commands/cx-spec.update.md",
    ".codex/skills/cx-spec.specify/SKILL.md",
    ".cx-spec/scripts/bash/update-cx-spec-kit.sh"
  ],
  "files_added": [
    ".cx-spec/templates/new-template.md"
  ],
  "message": "Updated from 1.0 to 1.1"
}
```
