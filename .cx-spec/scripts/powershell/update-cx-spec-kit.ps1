<#
.SYNOPSIS
    Update CX Spec Kit from GitHub repository.

.DESCRIPTION
    This script updates the local cx-spec-kit installation by fetching
    the latest version from the source GitHub repository.

.EXAMPLE
    .\update-cx-spec-kit.ps1

.NOTES
    Exit codes:
      0 - Success (updated or already up to date)
      1 - Error (network, parse, write failure)
#>

[CmdletBinding()]
param()

# Configuration
$script:GITHUB_OWNER = "CheckmarxDev"
$script:GITHUB_REPO = "internal-cx-agents"
$script:GITHUB_REF = "main"
$script:GITHUB_SUBDIR = "cx-spec-kit"
$script:GITHUB_RAW_BASE = "https://raw.githubusercontent.com/$($script:GITHUB_OWNER)/$($script:GITHUB_REPO)/$($script:GITHUB_REF)/$($script:GITHUB_SUBDIR)"
$script:GITHUB_API_BASE = "https://api.github.com/repos/$($script:GITHUB_OWNER)/$($script:GITHUB_REPO)/contents/$($script:GITHUB_SUBDIR)"
$script:GITHUB_API_ROOT = "repos/$($script:GITHUB_OWNER)/$($script:GITHUB_REPO)/contents/$($script:GITHUB_SUBDIR)"
$script:GitHubToken = $null
$script:GitHubTokenResolved = $false
$script:GhAuthChecked = $false
$script:GhAuthReady = $false
$script:FetchStrategyAnnounced = $false

# Global arrays to track updated and added files
$script:FilesUpdated = @()
$script:FilesAdded = @()
$script:FilesFailed = @()
$script:HadFailures = $false

# Output functions
function Write-Error-Message { param([string]$Message) Write-Host "❌ ERROR: $Message" -ForegroundColor Red }
function Write-Success { param([string]$Message) Write-Host "✅ $Message" -ForegroundColor Green }
function Write-Warning-Message { param([string]$Message) Write-Host "⚠️  $Message" -ForegroundColor Yellow }
function Write-Info { param([string]$Message) Write-Host "ℹ️  $Message" -ForegroundColor Cyan }
function Write-Step { param([string]$Message) Write-Host "🔄 $Message" -ForegroundColor Green }

# Reset cached auth/fetch state
function Reset-GitHubTokenCache {
    $script:GitHubToken = $null
    $script:GitHubTokenResolved = $false
    $script:GhAuthChecked = $false
    $script:GhAuthReady = $false
}

function Add-UpdateFailure {
    param([string]$Path)

    $script:HadFailures = $true
    $script:FilesFailed += $Path
}

# Resolve GitHub token from env only (strategy 2)
function Get-GitHubToken {
    if ($script:GitHubTokenResolved) {
        return $script:GitHubToken
    }

    $token = $null

    if ($env:GH_TOKEN) {
        $token = $env:GH_TOKEN
    } elseif ($env:GITHUB_TOKEN) {
        $token = $env:GITHUB_TOKEN
    }

    $script:GitHubToken = if ([string]::IsNullOrWhiteSpace($token)) { $null } else { $token }
    $script:GitHubTokenResolved = $true
    return $script:GitHubToken
}

# Check whether GitHub CLI auth is available (strategy 1)
function Test-GhCliAuthReady {
    if ($script:GhAuthChecked) {
        return $script:GhAuthReady
    }

    $script:GhAuthChecked = $true
    $gh = Get-Command gh -ErrorAction SilentlyContinue
    if (-not $gh) {
        $script:GhAuthReady = $false
        return $false
    }

    try {
        & gh auth status -h github.com *> $null
        $script:GhAuthReady = ($LASTEXITCODE -eq 0)
    } catch {
        $script:GhAuthReady = $false
    }

    return $script:GhAuthReady
}

# Announce chosen fetch strategy once
function Write-FetchStrategy {
    param([string]$Strategy)
    if ($script:FetchStrategyAnnounced) {
        return
    }
    Write-Info "Using GitHub fetch strategy: $Strategy"
    $script:FetchStrategyAnnounced = $true
}

# Keep for backward compatibility with callers.
# Update flow no longer requires preflight auth; fetch fallback handles it.
function Ensure-GitHubAuth {
    return $true
}

# Manual fallback guidance (strategy 4)
function Write-ManualFetchHelp {
    param([string]$Path)

    Write-Error-Message "Unable to fetch '$Path' after trying all automated strategies:"
    Write-Error-Message "1) GitHub CLI (gh auth)"
    Write-Error-Message "2) GH_TOKEN/GITHUB_TOKEN"
    Write-Error-Message "3) Public web fetch"
    Write-Error-Message "Manual fallback: https://github.com/$($script:GITHUB_OWNER)/$($script:GITHUB_REPO)/blob/$($script:GITHUB_REF)/$($script:GITHUB_SUBDIR)/$Path"
    Write-Error-Message "Or run: gh api repos/$($script:GITHUB_OWNER)/$($script:GITHUB_REPO)/contents/$($script:GITHUB_SUBDIR)/$Path --jq '.content' | base64 -d"
}

# Strategy 1: GitHub CLI
function Get-RemoteContentViaGhCli {
    param([string]$RemotePath)

    if (-not (Test-GhCliAuthReady)) {
        return $null
    }

    try {
        $encoded = (& gh api "$script:GITHUB_API_ROOT/$RemotePath?ref=$script:GITHUB_REF" --jq '.content' 2>$null | Out-String).Trim()
        if ([string]::IsNullOrWhiteSpace($encoded)) {
            return $null
        }

        $encoded = $encoded -replace '\r|\n', ''
        $bytes = [Convert]::FromBase64String($encoded)
        return [Text.Encoding]::UTF8.GetString($bytes)
    } catch {
        return $null
    }
}

function Get-RemoteFileListViaGhCli {
    param([string]$Path)

    if (-not (Test-GhCliAuthReady)) {
        return $null
    }

    try {
        $items = @(& gh api "$script:GITHUB_API_ROOT/$Path?ref=$script:GITHUB_REF" --jq '.[] | select(.type == "file") | .name' 2>$null)
        if ($LASTEXITCODE -ne 0) {
            return $null
        }

        return @($items | ForEach-Object { $_.ToString().Trim() } | Where-Object { $_ })
    } catch {
        return $null
    }
}

function ConvertTo-RemoteEntries {
    param([object]$Response)

    $entries = @()
    foreach ($item in @($Response)) {
        if ($null -eq $item) {
            continue
        }

        $entryType = [string]$item.type
        $entryName = [string]$item.name
        if ([string]::IsNullOrWhiteSpace($entryType) -or [string]::IsNullOrWhiteSpace($entryName)) {
            continue
        }

        if ($entryType -eq "file" -or $entryType -eq "dir") {
            $entries += [pscustomobject]@{
                type = $entryType
                name = $entryName
            }
        }
    }

    return $entries
}

function Get-RemoteEntriesViaGhCli {
    param([string]$Path)

    if (-not (Test-GhCliAuthReady)) {
        return $null
    }

    try {
        $raw = (& gh api "$script:GITHUB_API_ROOT/$Path?ref=$script:GITHUB_REF" 2>$null | Out-String).Trim()
        if ($LASTEXITCODE -ne 0) {
            return $null
        }

        if ([string]::IsNullOrWhiteSpace($raw)) {
            return @()
        }

        $response = $raw | ConvertFrom-Json
        return @(ConvertTo-RemoteEntries -Response $response)
    } catch {
        return $null
    }
}

function Save-RemoteFileViaGhCli {
    param(
        [string]$RemotePath,
        [string]$LocalPath
    )

    if (-not (Test-GhCliAuthReady)) {
        return $false
    }

    try {
        $encoded = (& gh api "$script:GITHUB_API_ROOT/$RemotePath?ref=$script:GITHUB_REF" --jq '.content' 2>$null | Out-String).Trim()
        if ([string]::IsNullOrWhiteSpace($encoded)) {
            return $false
        }

        $encoded = $encoded -replace '\r|\n', ''
        $bytes = [Convert]::FromBase64String($encoded)
        [IO.File]::WriteAllBytes($LocalPath, $bytes)
        return $true
    } catch {
        return $false
    }
}

# Strategy 2: GitHub API/raw with env token
function Get-RemoteContentViaToken {
    param([string]$RemotePath)

    $token = Get-GitHubToken
    if (-not $token) {
        return $null
    }

    try {
        return (Invoke-WebRequest -Uri "$script:GITHUB_RAW_BASE/$RemotePath" -Headers @{ Authorization = "Bearer $token" } -Method Get -ErrorAction Stop).Content
    } catch {
        return $null
    }
}

function Get-RemoteFileListViaToken {
    param([string]$Path)

    $token = Get-GitHubToken
    if (-not $token) {
        return $null
    }

    try {
        $response = Invoke-RestMethod -Uri "$script:GITHUB_API_BASE/$Path?ref=$script:GITHUB_REF" -Method Get -Headers @{ Authorization = "Bearer $token"; Accept = "application/vnd.github+json" } -ErrorAction Stop
        return @($response | Where-Object { $_.type -eq "file" } | Select-Object -ExpandProperty name)
    } catch {
        return $null
    }
}

function Get-RemoteEntriesViaToken {
    param([string]$Path)

    $token = Get-GitHubToken
    if (-not $token) {
        return $null
    }

    try {
        $response = Invoke-RestMethod -Uri "$script:GITHUB_API_BASE/$Path?ref=$script:GITHUB_REF" -Method Get -Headers @{ Authorization = "Bearer $token"; Accept = "application/vnd.github+json" } -ErrorAction Stop
        return @(ConvertTo-RemoteEntries -Response $response)
    } catch {
        return $null
    }
}

function Save-RemoteFileViaToken {
    param(
        [string]$RemotePath,
        [string]$LocalPath
    )

    $token = Get-GitHubToken
    if (-not $token) {
        return $false
    }

    try {
        Invoke-WebRequest -Uri "$script:GITHUB_RAW_BASE/$RemotePath" -Headers @{ Authorization = "Bearer $token" } -OutFile $LocalPath -ErrorAction Stop
        return $true
    } catch {
        return $false
    }
}

# Strategy 3: Public web fetch
function Get-RemoteContentViaWeb {
    param([string]$RemotePath)

    try {
        return (Invoke-WebRequest -Uri "$script:GITHUB_RAW_BASE/$RemotePath" -Method Get -ErrorAction Stop).Content
    } catch {
        return $null
    }
}

function Get-RemoteFileListViaWeb {
    param([string]$Path)

    try {
        $response = Invoke-RestMethod -Uri "$script:GITHUB_API_BASE/$Path?ref=$script:GITHUB_REF" -Method Get -ErrorAction Stop
        return @($response | Where-Object { $_.type -eq "file" } | Select-Object -ExpandProperty name)
    } catch {
        return $null
    }
}

function Get-RemoteEntriesViaWeb {
    param([string]$Path)

    try {
        $response = Invoke-RestMethod -Uri "$script:GITHUB_API_BASE/$Path?ref=$script:GITHUB_REF" -Method Get -ErrorAction Stop
        return @(ConvertTo-RemoteEntries -Response $response)
    } catch {
        return $null
    }
}

function Save-RemoteFileViaWeb {
    param(
        [string]$RemotePath,
        [string]$LocalPath
    )

    try {
        Invoke-WebRequest -Uri "$script:GITHUB_RAW_BASE/$RemotePath" -OutFile $LocalPath -ErrorAction Stop
        return $true
    } catch {
        return $false
    }
}

# Strategy orchestrators
function Get-RemoteContent {
    param(
        [string]$RemotePath,
        [switch]$SuppressManualHelp
    )

    $content = Get-RemoteContentViaGhCli -RemotePath $RemotePath
    if ($null -ne $content) {
        Write-FetchStrategy "GitHub CLI (gh auth)"
        return $content
    }

    $content = Get-RemoteContentViaToken -RemotePath $RemotePath
    if ($null -ne $content) {
        Write-FetchStrategy "GH_TOKEN/GITHUB_TOKEN"
        return $content
    }

    $content = Get-RemoteContentViaWeb -RemotePath $RemotePath
    if ($null -ne $content) {
        Write-FetchStrategy "Public web fetch"
        return $content
    }

    if (-not $SuppressManualHelp) {
        Write-ManualFetchHelp -Path $RemotePath
    }

    return $null
}

function Get-RemoteFileList {
    param([string]$Path)

    $files = Get-RemoteFileListRecursive -Path $Path -Prefix ""
    if ($null -eq $files) {
        return $null
    }

    return @($files)
}

function Get-RemoteEntries {
    param([string]$Path)

    $entries = Get-RemoteEntriesViaGhCli -Path $Path
    if ($null -ne $entries) {
        Write-FetchStrategy "GitHub CLI (gh auth)"
        return $entries
    }

    $entries = Get-RemoteEntriesViaToken -Path $Path
    if ($null -ne $entries) {
        Write-FetchStrategy "GH_TOKEN/GITHUB_TOKEN"
        return $entries
    }

    $entries = Get-RemoteEntriesViaWeb -Path $Path
    if ($null -ne $entries) {
        Write-FetchStrategy "Public web fetch"
        return $entries
    }

    Write-ManualFetchHelp -Path $Path
    return $null
}

function Get-RemoteFileListRecursive {
    param(
        [string]$Path,
        [string]$Prefix = ""
    )

    $entries = Get-RemoteEntries -Path $Path
    if ($null -eq $entries) {
        return $null
    }

    $files = @()
    foreach ($entry in @($entries)) {
        $entryType = [string]$entry.type
        $entryName = [string]$entry.name
        if ([string]::IsNullOrWhiteSpace($entryType) -or [string]::IsNullOrWhiteSpace($entryName)) {
            continue
        }

        if ($entryType -eq "file") {
            if ([string]::IsNullOrWhiteSpace($Prefix)) {
                $files += $entryName
            } else {
                $files += "$Prefix/$entryName"
            }
            continue
        }

        if ($entryType -eq "dir") {
            $childPath = "$Path/$entryName"
            $childPrefix = if ([string]::IsNullOrWhiteSpace($Prefix)) { $entryName } else { "$Prefix/$entryName" }
            $nestedFiles = Get-RemoteFileListRecursive -Path $childPath -Prefix $childPrefix
            if ($null -eq $nestedFiles) {
                return $null
            }
            $files += @($nestedFiles)
        }
    }

    return $files
}

# Download and save a file
function Save-RemoteFile {
    param(
        [string]$RemotePath,
        [string]$LocalPath
    )

    # Create parent directory if needed
    $parentDir = Split-Path -Parent $LocalPath
    if (-not (Test-Path $parentDir)) {
        New-Item -ItemType Directory -Path $parentDir -Force | Out-Null
    }

    if (Save-RemoteFileViaGhCli -RemotePath $RemotePath -LocalPath $LocalPath) {
        Write-FetchStrategy "GitHub CLI (gh auth)"
        return $true
    }

    if (Save-RemoteFileViaToken -RemotePath $RemotePath -LocalPath $LocalPath) {
        Write-FetchStrategy "GH_TOKEN/GITHUB_TOKEN"
        return $true
    }

    if (Save-RemoteFileViaWeb -RemotePath $RemotePath -LocalPath $LocalPath) {
        Write-FetchStrategy "Public web fetch"
        return $true
    }

    Write-ManualFetchHelp -Path $RemotePath
    return $false
}

# Get repository root
function Get-RepoRoot {
    try {
        $gitRoot = git rev-parse --show-toplevel 2>$null
        if ($LASTEXITCODE -eq 0 -and $gitRoot) {
            return $gitRoot.Trim()
        }
    } catch {}

    # Fall back to script location (deployed at <repo>/.cx-spec/scripts/powershell)
    return (Resolve-Path (Join-Path $PSScriptRoot "..\..\..")).Path
}

# Compare semantic versions
# Returns: 1 if v1 > v2, 0 if v1 = v2, -1 if v1 < v2
function Compare-SemanticVersions {
    param(
        [string]$Version1,
        [string]$Version2
    )

    if ([string]::IsNullOrEmpty($Version1)) { $Version1 = "0" }
    if ([string]::IsNullOrEmpty($Version2)) { $Version2 = "0" }

    $v1Parts = $Version1.Split('.') | ForEach-Object { [int]$_ }
    $v2Parts = $Version2.Split('.') | ForEach-Object { [int]$_ }

    $maxParts = [Math]::Max($v1Parts.Count, $v2Parts.Count)

    for ($i = 0; $i -lt $maxParts; $i++) {
        $p1 = if ($i -lt $v1Parts.Count) { $v1Parts[$i] } else { 0 }
        $p2 = if ($i -lt $v2Parts.Count) { $v2Parts[$i] } else { 0 }

        if ($p1 -gt $p2) { return 1 }
        if ($p1 -lt $p2) { return -1 }
    }

    return 0
}

# Fetch remote config and extract version
function Get-RemoteVersion {
    $remoteConfigText = Get-RemoteContent -RemotePath "config.json"
    if (-not $remoteConfigText) {
        Write-Error-Message "Failed to fetch remote config"
        return $null
    }

    try {
        $remoteConfig = $remoteConfigText | ConvertFrom-Json
        return $remoteConfig.version
    } catch {
        Write-Error-Message "Could not parse version from remote config"
        return $null
    }
}

# Get local version from config
function Get-LocalVersion {
    param([string]$RepoRoot)

    $configFile = Join-Path $RepoRoot ".cx-spec\config.json"

    if (-not (Test-Path $configFile)) {
        return $null
    }

    try {
        $config = Get-Content $configFile -Raw | ConvertFrom-Json
        return $config.version
    } catch {
        return $null
    }
}

# Returns true when value is a JSON-like object node (not array/scalar).
function Test-IsJsonObject {
    param([AllowNull()]$Value)

    if ($null -eq $Value) {
        return $false
    }

    return ($Value -is [System.Management.Automation.PSCustomObject]) -or ($Value -is [System.Collections.IDictionary])
}

# Deep-merge JSON objects using "remote base + local override" semantics.
# Arrays/scalars from local replace remote values. Object nodes merge recursively.
function Merge-DeepJsonObjects {
    param(
        [Parameter(Mandatory = $true)]$RemoteObject,
        [Parameter(Mandatory = $true)]$LocalObject
    )

    $merged = $RemoteObject | ConvertTo-Json -Depth 30 | ConvertFrom-Json

    foreach ($localProp in $LocalObject.PSObject.Properties) {
        $name = $localProp.Name
        $localValue = $localProp.Value
        if ($null -eq $localValue) {
            continue
        }

        $mergedProp = $merged.PSObject.Properties[$name]
        if ($null -eq $mergedProp) {
            $merged | Add-Member -NotePropertyName $name -NotePropertyValue $localValue -Force
            continue
        }

        $remoteValue = $mergedProp.Value
        if ((Test-IsJsonObject -Value $remoteValue) -and (Test-IsJsonObject -Value $localValue)) {
            $merged.$name = Merge-DeepJsonObjects -RemoteObject $remoteValue -LocalObject $localValue
        } else {
            $merged.$name = $localValue
        }
    }

    return $merged
}

# Merge configs: preserve user settings, update version and add new keys
function Merge-Configs {
    param(
        [string]$RepoRoot,
        [PSObject]$RemoteConfig
    )

    $localConfigFile = Join-Path $RepoRoot ".cx-spec\config.json"

    if (-not (Test-Path $localConfigFile)) {
        try {
            # No local config, just use remote
            $RemoteConfig | ConvertTo-Json -Depth 10 | Set-Content $localConfigFile -Encoding UTF8
            return $true
        } catch {
            return $false
        }
    }

    try {
        $localConfig = Get-Content $localConfigFile -Raw | ConvertFrom-Json

        # Merge: remote as base, preserve user-customized fields from local.
        # Use JSON round-trip for a reliable deep clone across PowerShell versions.
        $merged = $RemoteConfig | ConvertTo-Json -Depth 20 | ConvertFrom-Json

        # Preserve user settings without relying on missing-property behavior.
        $preservedKeys = @("workflow", "options", "mode_defaults", "spec_sync", "team_directives", "architecture")
        foreach ($key in $preservedKeys) {
            $localProp = $localConfig.PSObject.Properties[$key]
            if ($null -ne $localProp) {
                $localValue = $localProp.Value
                if ($null -eq $localValue) {
                    continue
                }

                if ($null -ne $merged.PSObject.Properties[$key]) {
                    $remoteValue = $merged.$key
                    if ((Test-IsJsonObject -Value $remoteValue) -and (Test-IsJsonObject -Value $localValue)) {
                        $merged.$key = Merge-DeepJsonObjects -RemoteObject $remoteValue -LocalObject $localValue
                    } else {
                        $merged.$key = $localValue
                    }
                } else {
                    $merged | Add-Member -NotePropertyName $key -NotePropertyValue $localValue -Force
                }
            }
        }

        # Version always comes from remote
        if ($null -ne $merged.PSObject.Properties["version"]) {
            $merged.version = $RemoteConfig.version
        } else {
            $merged | Add-Member -NotePropertyName "version" -NotePropertyValue $RemoteConfig.version -Force
        }

        $merged | ConvertTo-Json -Depth 10 | Set-Content $localConfigFile -Encoding UTF8
        return $true
    } catch {
        Write-Warning-Message "Failed to merge configs, using remote config"
        try {
            $RemoteConfig | ConvertTo-Json -Depth 10 | Set-Content $localConfigFile -Encoding UTF8
            return $true
        } catch {
            return $false
        }
    }
}

# Extract command description from frontmatter
function Get-CommandDescriptionFromFile {
    param([string]$SourceFile)

    try {
        $content = Get-Content -Path $SourceFile -Raw -ErrorAction Stop
    } catch {
        return "CX Spec Kit workflow skill"
    }

    if ($content -match '(?ms)^---\s*\r?\n(.*?)\r?\n---') {
        $frontmatter = $Matches[1]
        if ($frontmatter -match '(?m)^description:\s*(.+)$') {
            return $Matches[1].Trim()
        }
    }

    return "CX Spec Kit workflow skill"
}

function Get-CommandBodyWithoutFrontmatter {
    param([string]$SourceFile)

    $content = Get-Content -Path $SourceFile -Raw -ErrorAction Stop
    if ($content -match '(?ms)^---\s*\r?\n.*?\r?\n---\s*\r?\n?(.*)$') {
        return $Matches[1]
    }

    return $content
}

function Convert-CommandToCodexSkill {
    param(
        [string]$SourceFile,
        [string]$TargetFile,
        [string]$SkillName
    )

    $description = Get-CommandDescriptionFromFile -SourceFile $SourceFile
    $body = Get-CommandBodyWithoutFrontmatter -SourceFile $SourceFile

    $targetDir = Split-Path -Parent $TargetFile
    if (-not (Test-Path $targetDir)) {
        New-Item -ItemType Directory -Path $targetDir -Force | Out-Null
    }

    $skillContent = @(
        "---"
        "name: $SkillName"
        "description: >"
        "  $description"
        "---"
        ""
        "## Argument Handling"
        ""
        ('- Invocation format: `{0} <arguments>`' -f $SkillName)
        '- Treat text after the skill name as `$ARGUMENTS`.'
        '- If no arguments are provided, follow the empty-input behavior already documented below.'
        ""
        $body
    ) -join "`n"

    Set-Content -Path $TargetFile -Value $skillContent -Encoding UTF8
}

# Update command assets for a specific AI target
function Update-Commands {
    param(
        [string]$RepoRoot,
        [string]$Target
    )

    $targetDir = ""
    $targetDisplay = ""
    $updated = 0
    $added = 0

    if ($Target -eq "codex") {
        $targetDir = Join-Path $RepoRoot ".codex\skills"
        $targetDisplay = ".codex/skills"
        Write-Step "Updating codex skills..."
    } else {
        $targetDir = Join-Path $RepoRoot ".$Target\commands"
        $targetDisplay = ".$Target/commands"
        Write-Step "Updating $Target commands..."
    }

    $files = Get-RemoteFileList -Path "commands"
    if ($null -eq $files) {
        Write-Warning-Message "Failed to fetch command file list"
        Add-UpdateFailure -Path "$targetDisplay/*"
        return @{ Updated = 0; Added = 0 }
    }

    foreach ($file in $files) {
        if ($file -notlike "*.md") { continue }

        if ($Target -eq "codex") {
            $skillName = ($file -replace '\.md$', '') -replace '[\\/]+', '.'
            $localFile = Join-Path $targetDir "$skillName\SKILL.md"
            $displayPath = ".codex/skills/$skillName/SKILL.md"
        } else {
            $localFile = Join-Path $targetDir $file
            $displayPath = ".$Target/commands/$file"
        }
        $existed = Test-Path $localFile

        if ($Target -eq "codex") {
            $tempFile = Join-Path ([System.IO.Path]::GetTempPath()) ("cx-spec-update-" + [Guid]::NewGuid().ToString() + ".md")
            try {
                if (-not (Save-RemoteFile -RemotePath "commands/$file" -LocalPath $tempFile)) {
                    Write-Warning-Message "Failed to download $file"
                    Add-UpdateFailure -Path $displayPath
                    continue
                }

                try {
                    Convert-CommandToCodexSkill -SourceFile $tempFile -TargetFile $localFile -SkillName $skillName
                } catch {
                    Write-Warning-Message "Failed to generate codex skill for $file"
                    Add-UpdateFailure -Path $displayPath
                    continue
                }
            } finally {
                Remove-Item -Path $tempFile -ErrorAction SilentlyContinue
            }
        } else {
            if (-not (Save-RemoteFile -RemotePath "commands/$file" -LocalPath $localFile)) {
                Write-Warning-Message "Failed to download $file"
                Add-UpdateFailure -Path $displayPath
                continue
            }
        }

        if ($existed) {
            $updated++
            $script:FilesUpdated += $displayPath
        } else {
            $added++
            $script:FilesAdded += $displayPath
        }
    }

    return @{ Updated = $updated; Added = $added }
}

# Update scripts (bash and powershell)
function Update-Scripts {
    param([string]$RepoRoot)

    $updated = 0
    $added = 0

    Write-Step "Updating scripts..."

    # Update bash scripts
    $bashFiles = Get-RemoteFileList -Path "scripts/bash"
    if ($null -eq $bashFiles) {
        Write-Warning-Message "Failed to fetch bash script file list"
        Add-UpdateFailure -Path ".cx-spec/scripts/bash/*"
        return @{ Updated = 0; Added = 0 }
    }
    foreach ($file in $bashFiles) {
        if ($file -notlike "*.sh") { continue }

        $localFile = Join-Path $RepoRoot ".cx-spec\scripts\bash\$file"
        $displayPath = ".cx-spec/scripts/bash/$file"
        $existed = Test-Path $localFile

        if (-not (Save-RemoteFile -RemotePath "scripts/bash/$file" -LocalPath $localFile)) {
            Write-Warning-Message "Failed to download bash/$file"
            Add-UpdateFailure -Path $displayPath
            continue
        }

        if ($existed) {
            $updated++
            $script:FilesUpdated += $displayPath
        } else {
            $added++
            $script:FilesAdded += $displayPath
        }
    }

    # Update powershell scripts
    $psFiles = Get-RemoteFileList -Path "scripts/powershell"
    if ($null -eq $psFiles) {
        Write-Warning-Message "Failed to fetch PowerShell script file list"
        Add-UpdateFailure -Path ".cx-spec/scripts/powershell/*"
        return @{ Updated = $updated; Added = $added }
    }
    foreach ($file in $psFiles) {
        if ($file -notlike "*.ps1") { continue }

        $localFile = Join-Path $RepoRoot ".cx-spec\scripts\powershell\$file"
        $displayPath = ".cx-spec/scripts/powershell/$file"
        $existed = Test-Path $localFile

        if (-not (Save-RemoteFile -RemotePath "scripts/powershell/$file" -LocalPath $localFile)) {
            Write-Warning-Message "Failed to download powershell/$file"
            Add-UpdateFailure -Path $displayPath
            continue
        }

        if ($existed) {
            $updated++
            $script:FilesUpdated += $displayPath
        } else {
            $added++
            $script:FilesAdded += $displayPath
        }
    }

    return @{ Updated = $updated; Added = $added }
}

# Update templates
function Update-Templates {
    param([string]$RepoRoot)

    $updated = 0
    $added = 0

    Write-Step "Updating templates..."

    $files = Get-RemoteFileList -Path "templates"
    if ($null -eq $files) {
        Write-Warning-Message "Failed to fetch template file list"
        Add-UpdateFailure -Path ".cx-spec/templates/*"
        return @{ Updated = 0; Added = 0 }
    }

    foreach ($file in $files) {
        if ($file -notlike "*.md") { continue }

        $localFile = Join-Path $RepoRoot ".cx-spec\templates\$file"
        $displayPath = ".cx-spec/templates/$file"
        $existed = Test-Path $localFile

        if (-not (Save-RemoteFile -RemotePath "templates/$file" -LocalPath $localFile)) {
            Write-Warning-Message "Failed to download template $file"
            Add-UpdateFailure -Path $displayPath
            continue
        }

        if ($existed) {
            $updated++
            $script:FilesUpdated += $displayPath
        } else {
            $added++
            $script:FilesAdded += $displayPath
        }
    }

    return @{ Updated = $updated; Added = $added }
}

# Detect which AI target directories exist
function Get-AITargets {
    param([string]$RepoRoot)

    $targets = @()

    if (Test-Path (Join-Path $RepoRoot ".cursor\commands")) { $targets += "cursor" }
    if (Test-Path (Join-Path $RepoRoot ".claude\commands")) { $targets += "claude" }
    if (Test-Path (Join-Path $RepoRoot ".codex\skills")) { $targets += "codex" }

    return $targets
}

# Output JSON result
function Write-Result {
    param(
        [string]$Status,
        [string]$LocalVersion,
        [string]$RemoteVersion,
        [int]$FilesUpdatedCount,
        [int]$FilesAddedCount,
        [int]$FilesFailedCount = $script:FilesFailed.Count,
        [string]$Message
    )

    @{
        status = $Status
        local_version = $LocalVersion
        remote_version = $RemoteVersion
        files_updated_count = $FilesUpdatedCount
        files_added_count = $FilesAddedCount
        files_failed_count = $FilesFailedCount
        files_updated = $script:FilesUpdated
        files_added = $script:FilesAdded
        files_failed = $script:FilesFailed
        message = $Message
    } | ConvertTo-Json -Depth 3
}

# Main function
function Main {
    Write-Host ""
    Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    Write-Host "🔄 CX Spec Kit Update"
    Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    Write-Host ""

    $script:FilesUpdated = @()
    $script:FilesAdded = @()
    $script:FilesFailed = @()
    $script:HadFailures = $false
    $script:FetchStrategyAnnounced = $false
    Reset-GitHubTokenCache

    $repoRoot = Get-RepoRoot
    Write-Info "Repository root: $repoRoot"

    # Step 1: Fetch remote version
    Write-Step "Checking for updates..."

    $remoteVersion = Get-RemoteVersion
    if (-not $remoteVersion) {
        Write-Error-Message "Failed to check for updates"
        Write-Result -Status "error" -LocalVersion "" -RemoteVersion "" -FilesUpdatedCount 0 -FilesAddedCount 0 -FilesFailedCount 0 -Message "Failed to fetch remote version"
        exit 1
    }

    # Step 2: Get local version
    $localVersion = Get-LocalVersion -RepoRoot $repoRoot

    Write-Info "Local version: $(if ($localVersion) { $localVersion } else { '(not installed)' })"
    Write-Info "Remote version: $remoteVersion"

    # Step 3: Compare versions
    if ($localVersion) {
        $cmpResult = Compare-SemanticVersions -Version1 $remoteVersion -Version2 $localVersion

        if ($cmpResult -le 0) {
            Write-Success "Already up to date (v$localVersion)"
            Write-Host ""
            Write-Result -Status "up_to_date" -LocalVersion $localVersion -RemoteVersion $remoteVersion -FilesUpdatedCount 0 -FilesAddedCount 0 -FilesFailedCount 0 -Message "Already up to date"
            exit 0
        }
    }

    # Step 4: Perform update
    Write-Host ""
    $fromVersion = if ($localVersion) { $localVersion } else { "(none)" }
    Write-Info "Updating from $fromVersion to $remoteVersion..."
    Write-Host ""

    $totalUpdated = 0
    $totalAdded = 0

    # Update scripts
    $scriptResult = Update-Scripts -RepoRoot $repoRoot
    $totalUpdated += $scriptResult.Updated
    $totalAdded += $scriptResult.Added

    # Update templates
    $templateResult = Update-Templates -RepoRoot $repoRoot
    $totalUpdated += $templateResult.Updated
    $totalAdded += $templateResult.Added

    # Update commands for each detected AI target
    $aiTargets = Get-AITargets -RepoRoot $repoRoot

    foreach ($target in $aiTargets) {
        $cmdResult = Update-Commands -RepoRoot $repoRoot -Target $target
        $totalUpdated += $cmdResult.Updated
        $totalAdded += $cmdResult.Added
    }

    # Update config (merge)
    if ($script:HadFailures) {
        Write-Warning-Message "Skipping configuration update because earlier update steps failed"
    } else {
        Write-Step "Updating configuration..."
        $remoteConfigText = Get-RemoteContent -RemotePath "config.json" -SuppressManualHelp
        if (-not $remoteConfigText) {
            Write-Warning-Message "Failed to update configuration"
            Add-UpdateFailure -Path ".cx-spec/config.json"
        } else {
            try {
                $remoteConfig = $remoteConfigText | ConvertFrom-Json
                if (-not (Merge-Configs -RepoRoot $repoRoot -RemoteConfig $remoteConfig)) {
                    Write-Warning-Message "Failed to merge configuration"
                    Add-UpdateFailure -Path ".cx-spec/config.json"
                }
            } catch {
                Write-Warning-Message "Failed to parse remote configuration"
                Add-UpdateFailure -Path ".cx-spec/config.json"
            }
        }
    }

    $totalFailed = $script:FilesFailed.Count

    # Summary
    Write-Host ""
    Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    if ($script:HadFailures) {
        Write-Error-Message "Update finished with errors"
    } else {
        Write-Success "Update complete!"
    }
    Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    Write-Info "Version: $fromVersion → $remoteVersion"
    Write-Host ""

    # Print updated files
    if ($script:FilesUpdated.Count -gt 0) {
        Write-Host "📝 Files updated ($($script:FilesUpdated.Count)):" -ForegroundColor Blue
        foreach ($file in $script:FilesUpdated) {
            Write-Host "   • $file"
        }
        Write-Host ""
    }

    # Print added files
    if ($script:FilesAdded.Count -gt 0) {
        Write-Host "✨ Files added ($($script:FilesAdded.Count)):" -ForegroundColor Green
        foreach ($file in $script:FilesAdded) {
            Write-Host "   • $file"
        }
        Write-Host ""
    }

    # Print failed files
    if ($script:FilesFailed.Count -gt 0) {
        Write-Host "❌ Files failed ($($script:FilesFailed.Count)):" -ForegroundColor Red
        foreach ($file in $script:FilesFailed) {
            Write-Host "   • $file"
        }
        Write-Host ""
    }

    if ($script:HadFailures) {
        Write-Result -Status "error" -LocalVersion $localVersion -RemoteVersion $remoteVersion -FilesUpdatedCount $totalUpdated -FilesAddedCount $totalAdded -FilesFailedCount $totalFailed -Message "Update failed before completion; local version was kept at $fromVersion."
        exit 1
    }

    Write-Result -Status "updated" -LocalVersion $localVersion -RemoteVersion $remoteVersion -FilesUpdatedCount $totalUpdated -FilesAddedCount $totalAdded -FilesFailedCount $totalFailed -Message "Updated from $fromVersion to $remoteVersion"
}

# Run main when executed directly; skip on dot-source to enable unit tests.
$invocationLine = if ($null -ne $MyInvocation.Line) { $MyInvocation.Line.TrimStart() } else { "" }
$isDotSourced = ($MyInvocation.InvocationName -eq '.') -or $invocationLine.StartsWith('. ')
$skipMainForTests = ($env:CX_SPEC_UPDATE_TEST_MODE -eq '1')
if (-not $isDotSourced -and -not $skipMainForTests) {
    Main
}
