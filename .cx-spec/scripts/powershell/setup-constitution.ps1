# Setup project constitution with team inheritance
[CmdletBinding()]
param(
    [switch]$Json,
    [switch]$Validate,
    [switch]$Scan,
    [switch]$Help
)

$ErrorActionPreference = 'Stop'

function Get-ProjectConfigPath {
    $repoRoot = Get-RepositoryRoot
    return Join-Path $repoRoot ".cx-spec\config.json"
}

function Get-ConfigPath {
    return (Get-ProjectConfigPath)
}

if ($Help) {
    Write-Output "Usage: ./setup-constitution.ps1 [-Json] [-Validate] [-Scan] [-Help]"
    Write-Output "  -Json     Output results in JSON format"
    Write-Output "  -Validate Validate existing constitution against team inheritance"
    Write-Output "  -Scan     Scan project artifacts and suggest constitution enhancements"
    Write-Output "  -Help     Show this help message"
    exit 0
}

# Get repository root
function Get-RepositoryRoot {
    # Try environment variable first
    if ($env:REPO_ROOT -and (Test-Path $env:REPO_ROOT)) {
        return $env:REPO_ROOT
    }

    # Try git root if available
    try {
        $gitRoot = git rev-parse --show-toplevel 2>$null
        if ($LASTEXITCODE -eq 0 -and $gitRoot -and (Test-Path $gitRoot.Trim())) {
            return $gitRoot.Trim()
        }
    } catch {
        # Ignore and continue to marker search
    }

    # Fallback: search for repository markers
    $currentDir = Get-Location
    while ($currentDir -and (Test-Path $currentDir)) {
        if ((Test-Path "$currentDir\.git") -or (Test-Path "$currentDir\.cx-spec")) {
            return $currentDir
        }
        $currentDir = Split-Path $currentDir -Parent
    }

    throw "Could not determine repository root"
}

# Get team directives path (from config)
function Get-TeamDirectivesPath {
    $repoRoot = Get-RepositoryRoot
    $projectConfig = Get-ProjectConfigPath
    $candidates = @()
    if (Test-Path $projectConfig) {
        $candidates += $projectConfig
    }

    foreach ($configFile in $candidates) {
        try {
            $config = Get-Content $configFile -Raw | ConvertFrom-Json
            $path = $config.team_directives.path
            if ($path) {
                $path = $path.Trim()
                if (-not [System.IO.Path]::IsPathRooted($path)) {
                    $path = Join-Path $repoRoot $path
                }
                if (Test-Path $path) {
                    return $path
                }
            }
        } catch {
            # Ignore config parsing errors
        }
    }

    return $null
}

# Load team constitution
function Get-TeamConstitution {
    $teamDirectives = Get-TeamDirectivesPath

    if ($teamDirectives -and (Test-Path $teamDirectives)) {
        # Try direct constitution.md
        $constitutionFile = Join-Path $teamDirectives "constitution.md"
        if (Test-Path $constitutionFile) {
            return Get-Content $constitutionFile -Raw
        }

        # Try context_modules/constitution.md
        $constitutionFile = Join-Path $teamDirectives "context_modules\constitution.md"
        if (Test-Path $constitutionFile) {
            return Get-Content $constitutionFile -Raw
        }
    }

    # Default constitution if none found
    return @"
# Project Constitution

## Core Principles

### Principle 1: Quality First
All code must meet quality standards and include appropriate testing.

### Principle 2: Documentation Required
Clear documentation must accompany all significant changes.

### Principle 3: Security by Default
Security considerations must be addressed for all features.

## Governance

**Version**: 1.0.0 | **Ratified**: $(Get-Date -Format 'yyyy-MM-dd') | **Last Amended**: $(Get-Date -Format 'yyyy-MM-dd')

*This constitution was auto-generated from team defaults. Customize as needed for your project.*
"@
}

# Enhance constitution with project context
function Add-ProjectContext {
    param([string]$Constitution)

    $repoRoot = Get-RepositoryRoot
    $projectName = Split-Path $repoRoot -Leaf

    # Add project header if not present
    if ($Constitution -notmatch "^# $projectName Constitution") {
        $inheritanceDate = Get-Date -Format 'yyyy-MM-dd'
        $Constitution = "# $projectName Constitution

*Inherited from team constitution on $inheritanceDate*

$Constitution"
    }

    return $Constitution
}

# Validate inheritance
function Test-ConstitutionInheritance {
    param([string]$TeamConstitution, [string]$ProjectConstitution)

    # Extract team principles (simple pattern match)
    $teamPrinciples = @()
    if ($TeamConstitution -match '^\d+\.\s*\*\*(.+?)\*\*') {
        $matches = [regex]::Matches($TeamConstitution, '^\d+\.\s*\*\*(.+?)\*\*', [System.Text.RegularExpressions.RegexOptions]::Multiline)
        foreach ($match in $matches) {
            $teamPrinciples += $match.Groups[1].Value
        }
    }

    # Check if project contains team principles
    $missingPrinciples = @()
    foreach ($principle in $teamPrinciples) {
        if ($ProjectConstitution -notmatch [regex]::Escape($principle)) {
            $missingPrinciples += $principle
        }
    }

    if ($missingPrinciples.Count -gt 0) {
        Write-Warning "Project constitution may be missing some team principles: $($missingPrinciples -join ', ')"
        Write-Host "Consider ensuring all team principles are represented in your project constitution."
        return $false
    } else {
        Write-Host "✓ Inheritance validation passed - all team principles detected in project constitution"
        return $true
    }
}

# Check for team constitution updates
function Test-TeamConstitutionUpdates {
    param([string]$TeamConstitution, [string]$ProjectConstitution)

    if ($ProjectConstitution -match 'Inherited from team constitution on (\d{4}-\d{2}-\d{2})') {
        $inheritanceDate = $matches[1]

        $teamDirectives = Get-TeamDirectivesPath
        if ($teamDirectives) {
            $constitutionFile = Join-Path $teamDirectives "context_modules\constitution.md"
            if (-not (Test-Path $constitutionFile)) {
                $constitutionFile = Join-Path $teamDirectives "constitution.md"
            }

            if (Test-Path $constitutionFile) {
                $teamFileInfo = Get-Item $constitutionFile
                $inheritanceDateTime = [DateTime]::Parse($inheritanceDate)

                if ($teamFileInfo.LastWriteTime -gt $inheritanceDateTime) {
                    Write-Host "NOTICE: Team constitution has been updated since project constitution was created."
                    Write-Host "Consider reviewing the team constitution for any changes that should be reflected in your project."
                    Write-Host "Team constitution: $constitutionFile"
                }
            }
        }
    }
}

function To-RuleStringArray {
    param($Value)

    if ($null -eq $Value) {
        return @()
    }

    if ($Value -is [string]) {
        if ([string]::IsNullOrWhiteSpace($Value)) {
            return @()
        }
        return @($Value.Trim())
    }

    if ($Value -is [System.Collections.IEnumerable]) {
        $items = @()
        foreach ($item in $Value) {
            if ($item -is [string] -and -not [string]::IsNullOrWhiteSpace($item)) {
                $items += $item.Trim()
            }
        }
        return $items
    }

    return @()
}

function To-RuleObjectArray {
    param($Value)

    if ($null -eq $Value) {
        return @()
    }

    if ($Value -is [System.Collections.IEnumerable] -and -not ($Value -is [string])) {
        return @($Value)
    }

    return @($Value)
}

function Resolve-RulePath {
    param(
        [string]$RawPath,
        [string]$RepoRoot
    )

    if ([string]::IsNullOrWhiteSpace($RawPath)) {
        return ""
    }

    if ([System.IO.Path]::IsPathRooted($RawPath)) {
        return $RawPath
    }

    return Join-Path $RepoRoot $RawPath
}

function Convert-RuleGlobToRegex {
    param([string]$Pattern)

    $normalized = ($Pattern -replace '\\', '/')
    $escaped = [regex]::Escape($normalized)
    $escaped = $escaped -replace '\\\*\\\*', '.*'
    $escaped = $escaped -replace '\\\*', '[^/]*'
    $escaped = $escaped -replace '\\\?', '[^/]'
    return "^$escaped$"
}

function Test-RuleGlobMatch {
    param(
        [string]$RelativePath,
        [string]$Pattern
    )

    if ([string]::IsNullOrWhiteSpace($Pattern)) {
        return $false
    }

    if ($Pattern -eq '**' -or $Pattern -eq '**/*' -or $Pattern -eq '*') {
        return $true
    }

    $normalizedPath = ($RelativePath -replace '\\', '/')
    $regexPattern = Convert-RuleGlobToRegex -Pattern $Pattern

    if ($normalizedPath -match $regexPattern) {
        return $true
    }

    if ($Pattern -notmatch '/') {
        $name = [System.IO.Path]::GetFileName($normalizedPath)
        if ($name -match $regexPattern) {
            return $true
        }
    }

    return $false
}

function Test-RulePathMatchesAny {
    param(
        [string]$RelativePath,
        [string[]]$Patterns
    )

    if ($null -eq $Patterns -or $Patterns.Count -eq 0) {
        return $true
    }

    foreach ($pattern in $Patterns) {
        if (Test-RuleGlobMatch -RelativePath $RelativePath -Pattern $pattern) {
            return $true
        }
    }

    return $false
}

function Get-RuleScanFiles {
    param(
        [string]$RepoRoot,
        [string[]]$IncludeAny,
        [string[]]$ExcludeAny,
        [hashtable]$ScanCache
    )

    $cacheKey = ($IncludeAny -join '|') + '::' + ($ExcludeAny -join '|')
    if ($ScanCache.Files.ContainsKey($cacheKey)) {
        return @($ScanCache.Files[$cacheKey])
    }

    $files = @()
    $repoRootPath = [System.IO.Path]::GetFullPath($RepoRoot)

    Get-ChildItem -Path $repoRootPath -Recurse -File -ErrorAction SilentlyContinue | ForEach-Object {
        $fullPath = $_.FullName
        if ($fullPath -match '[\\/]\\.git([\\/]|$)') {
            return
        }

        $relative = [System.IO.Path]::GetRelativePath($repoRootPath, $fullPath)
        $relative = $relative -replace '\\', '/'

        if (-not (Test-RulePathMatchesAny -RelativePath $relative -Patterns $IncludeAny)) {
            return
        }

        if (Test-RulePathMatchesAny -RelativePath $relative -Patterns $ExcludeAny) {
            return
        }

        $files += $_
    }

    $ScanCache.Files[$cacheKey] = $files
    return @($files)
}

function Get-RuleFileContent {
    param(
        [System.IO.FileInfo]$File,
        [hashtable]$ScanCache
    )

    $cacheKey = $File.FullName
    if ($ScanCache.Content.ContainsKey($cacheKey)) {
        return [string]$ScanCache.Content[$cacheKey]
    }

    $content = ""
    try {
        if ($File.Length -le 2MB) {
            $content = Get-Content -Path $File.FullName -Raw -ErrorAction SilentlyContinue
            if ($null -eq $content) {
                $content = ""
            }
        }
    } catch {
        $content = ""
    }

    $ScanCache.Content[$cacheKey] = $content
    return [string]$content
}

function Test-RuleRegexCondition {
    param(
        $Condition,
        [string]$RepoRoot,
        [hashtable]$ScanCache
    )

    $conditionType = [string]$Condition.type
    $includeAny = To-RuleStringArray $Condition.include_any
    $excludeAny = To-RuleStringArray $Condition.exclude_any
    $patternsAny = To-RuleStringArray $Condition.patterns_any

    if ($includeAny.Count -eq 0) {
        $includeAny = @('**/*')
    }

    if ($conditionType -ne 'regex_any' -and $conditionType -ne 'regex_all') {
        throw "Unsupported detection type '$conditionType'."
    }

    if ($patternsAny.Count -eq 0) {
        throw "Detection regex condition must define non-empty patterns_any."
    }

    $compiledPatterns = @()
    foreach ($pattern in $patternsAny) {
        $compiledPatterns += [regex]::new($pattern, [System.Text.RegularExpressions.RegexOptions]::Multiline)
    }

    $files = Get-RuleScanFiles -RepoRoot $RepoRoot -IncludeAny $includeAny -ExcludeAny $excludeAny -ScanCache $ScanCache
    if ($files.Count -eq 0) {
        return $false
    }

    if ($conditionType -eq 'regex_any') {
        foreach ($file in $files) {
            $content = Get-RuleFileContent -File $file -ScanCache $ScanCache
            if ([string]::IsNullOrEmpty($content)) {
                continue
            }

            foreach ($pattern in $compiledPatterns) {
                if ($pattern.IsMatch($content)) {
                    return $true
                }
            }
        }

        return $false
    }

    $remaining = New-Object System.Collections.Generic.HashSet[int]
    for ($i = 0; $i -lt $compiledPatterns.Count; $i++) {
        [void]$remaining.Add($i)
    }

    foreach ($file in $files) {
        if ($remaining.Count -eq 0) {
            break
        }

        $content = Get-RuleFileContent -File $file -ScanCache $ScanCache
        if ([string]::IsNullOrEmpty($content)) {
            continue
        }

        foreach ($index in @($remaining)) {
            if ($compiledPatterns[$index].IsMatch($content)) {
                [void]$remaining.Remove($index)
            }
        }
    }

    return ($remaining.Count -eq 0)
}

function Test-RuleDetection {
    param(
        $Detection,
        [string]$RepoRoot,
        [hashtable]$ScanCache
    )

    if ($null -eq $Detection) {
        return $false
    }

    $rulesAny = To-RuleObjectArray $Detection.rules_any
    if ($rulesAny.Count -gt 0 -and $null -ne $Detection.rules_any) {
        foreach ($condition in $rulesAny) {
            if ($null -eq $condition) {
                continue
            }
            if (Test-RuleRegexCondition -Condition $condition -RepoRoot $RepoRoot -ScanCache $ScanCache) {
                return $true
            }
        }
        return $false
    }

    $rulesAll = To-RuleObjectArray $Detection.rules_all
    if ($rulesAll.Count -gt 0 -and $null -ne $Detection.rules_all) {
        foreach ($condition in $rulesAll) {
            if ($null -eq $condition) {
                return $false
            }
            if (-not (Test-RuleRegexCondition -Condition $condition -RepoRoot $RepoRoot -ScanCache $ScanCache)) {
                return $false
            }
        }
        return $true
    }

    return $false
}

function Get-CustomConstitutionPrinciplesRegistryPath {
    param(
        [string]$RepoRoot,
        [string]$ScriptRoot
    )

    $candidates = @()

    if (-not [string]::IsNullOrWhiteSpace($env:CUSTOM_CONSTITUTION_PRINCIPLES_CONFIG)) {
        $candidates += Resolve-RulePath -RawPath $env:CUSTOM_CONSTITUTION_PRINCIPLES_CONFIG -RepoRoot $RepoRoot
    }

    $projectConfig = Join-Path $RepoRoot '.cx-spec/config.json'
    if (Test-Path $projectConfig -PathType Leaf) {
        try {
            $configPayload = Get-Content -Path $projectConfig -Raw | ConvertFrom-Json
            $configuredPath = [string]$configPayload.custom_constitution_principles.config
            if (-not [string]::IsNullOrWhiteSpace($configuredPath)) {
                $candidates += Resolve-RulePath -RawPath $configuredPath -RepoRoot $RepoRoot
            }
        } catch {
            # Ignore malformed config and keep fallback candidates.
        }
    }

    $candidates += Join-Path $RepoRoot '.cx-spec/templates/custom-constitution-principles.json'
    $candidates += Join-Path (Resolve-Path (Join-Path $ScriptRoot '../..')).Path 'templates/custom-constitution-principles.json'

    $seen = @{}
    foreach ($candidate in $candidates) {
        if ([string]::IsNullOrWhiteSpace($candidate)) {
            continue
        }
        if ($seen.ContainsKey($candidate)) {
            continue
        }
        $seen[$candidate] = $true

        if (Test-Path $candidate -PathType Leaf) {
            return $candidate
        }
    }

    return $null
}

function Inject-CustomPrinciples {
    param(
        [string]$Constitution,
        [string]$RepoRoot,
        [string]$ScriptRoot
    )

    $registryPath = Get-CustomConstitutionPrinciplesRegistryPath -RepoRoot $RepoRoot -ScriptRoot $ScriptRoot
    $placeholder = '[CUSTOM_CONSTITUTION_PRINCIPLES]'
    if (-not $registryPath) {
        $updated = $Constitution
        if ($updated.Contains($placeholder)) {
            $updated = $updated.Replace($placeholder, '')
        }
        return [PSCustomObject]@{
            Constitution = $updated
            InjectedPrincipleIds = @()
        }
    }

    $registry = Get-Content -Path $registryPath -Raw | ConvertFrom-Json
    $principles = To-RuleObjectArray $registry.principles
    if ($principles.Count -eq 0 -or $null -eq $registry.principles) {
        throw "Invalid principles registry format: $registryPath"
    }

    $scanCache = @{
        Files = @{}
        Content = @{}
    }

    $injectedPrincipleIds = @()
    $snippets = @()
    $registryDir = Split-Path $registryPath -Parent

    foreach ($principle in $principles) {
        if ($null -eq $principle) {
            continue
        }

        if ($null -ne $principle.enabled -and $principle.enabled -is [bool] -and -not [bool]$principle.enabled) {
            continue
        }

        $principleFile = [string]$principle.principle_file
        if ([string]::IsNullOrWhiteSpace($principleFile)) {
            continue
        }

        if (-not (Test-RuleDetection -Detection $principle.detection -RepoRoot $RepoRoot -ScanCache $scanCache)) {
            continue
        }

        $principlePath = $principleFile
        if (-not [System.IO.Path]::IsPathRooted($principlePath)) {
            $principlePath = Join-Path $registryDir $principlePath
        }

        if (-not (Test-Path $principlePath -PathType Leaf)) {
            throw "Principle file not found for principle '$([string]$principle.id)': $principlePath"
        }

        $principleContent = Get-Content -Path $principlePath -Raw
        $snippets += $principleContent.Trim()
        $injectedPrincipleIds += [string]$principle.id
    }

    if ($snippets.Count -eq 0) {
        $updated = $Constitution
        if ($updated.Contains($placeholder)) {
            $updated = $updated.Replace($placeholder, '')
        }
        return [PSCustomObject]@{
            Constitution = $updated
            InjectedPrincipleIds = @()
        }
    }

    $section = "## Custom Constitution Principles`n`n$($snippets -join "`n`n")`n`n"
    if ($Constitution.Contains($placeholder)) {
        $updated = $Constitution.Replace($placeholder, $section.TrimEnd())
    } else {
        $marker = '## Governance'
        $index = $Constitution.IndexOf($marker)
        if ($index -ge 0) {
            $updated = $Constitution.Substring(0, $index) + $section + $Constitution.Substring($index)
        } else {
            $updated = $Constitution.TrimEnd() + "`n`n" + $section.TrimEnd() + "`n"
        }
    }

    return [PSCustomObject]@{
        Constitution = $updated
        InjectedPrincipleIds = $injectedPrincipleIds
    }
}

# Main logic
try {
    $repoRoot = Get-RepositoryRoot
    $constitutionFile = Join-Path $repoRoot ".cx-spec\memory\constitution.md"

    # Ensure directory exists
    $constitutionDir = Split-Path $constitutionFile -Parent
    if (-not (Test-Path $constitutionDir)) {
        New-Item -ItemType Directory -Path $constitutionDir -Force | Out-Null
    }

    if ($Scan -and -not (Test-Path $constitutionFile)) {
        Write-Host "Scanning project artifacts for constitution suggestions..." -ForegroundColor Cyan
        & "$PSScriptRoot\scan-project-artifacts.ps1" -Suggestions
        exit 0
    }

    if ($Validate) {
        if (-not (Test-Path $constitutionFile)) {
            Write-Error "No constitution file found at $constitutionFile. Run without --validate to create the constitution first."
            exit 1
        }

        $teamConstitution = Get-TeamConstitution
        $projectConstitution = Get-Content $constitutionFile -Raw
        $teamDirectivesPath = Get-TeamDirectivesPath
        if (-not $teamDirectivesPath) { $teamDirectivesPath = "" }

        if ($Json) {
            $result = @{
                status = "validated"
                file = $constitutionFile
                team_directives = $teamDirectivesPath
            } | ConvertTo-Json -Compress
            Write-Host $result
        } else {
            Write-Host "Validating constitution at: $constitutionFile"
            Write-Host "Team directives source: $teamDirectivesPath"
            Write-Host ""
            Test-ConstitutionInheritance -TeamConstitution $teamConstitution -ProjectConstitution $projectConstitution
            Write-Host ""
            Test-TeamConstitutionUpdates -TeamConstitution $teamConstitution -ProjectConstitution $projectConstitution
        }
        exit 0
    }

    if (Test-Path $constitutionFile) {
        Write-Host "Constitution file already exists at $constitutionFile"
        Write-Host "Use git to modify it directly, or remove it to recreate from team directives."

        # Check for updates
        $teamConstitution = Get-TeamConstitution
        $existingConstitution = Get-Content $constitutionFile -Raw
        Test-TeamConstitutionUpdates -TeamConstitution $teamConstitution -ProjectConstitution $existingConstitution
        Write-Host ""

        if ($Json) {
            $result = @{ status = "exists"; file = $constitutionFile } | ConvertTo-Json -Compress
            Write-Host $result
        }
        exit 0
    }

    # Create new constitution
    $teamConstitution = Get-TeamConstitution
    $projectConstitution = Add-ProjectContext -Constitution $teamConstitution
    $injectedCustomPrinciples = @()

    # If scan mode is enabled, enhance constitution with project insights
    if ($Scan) {
        if (-not $Json) {
            Write-Host "Enhancing constitution with project artifact analysis..." -ForegroundColor Cyan
        }

        # Get scan results
        $scanResults = & "$PSScriptRoot\scan-project-artifacts.ps1" -Json | ConvertFrom-Json

        # Generate additional principles based on scan
        $additionalPrinciples = @()

        # Parse testing data
        $testingParts = $scanResults.testing -split '\|'
        $testFiles = [int]$testingParts[0]
        $testFrameworks = $testingParts[1]

        if ($testFiles -gt 0) {
            $additionalPrinciples += @"
### Tests Drive Confidence (Project Practice)
Automated testing is established with $testFiles test files using $testFrameworks. All features must maintain or improve test coverage. Refuse to ship when test suites fail.
"@
        }

        # Parse security data
        $securityParts = $scanResults.security -split '\|'
        $authPatterns = [int]$securityParts[0]
        $securityIndicators = [int]$securityParts[2]

        if ($authPatterns -gt 0 -or $securityIndicators -gt 0) {
            $additionalPrinciples += @"
### Security by Default (Project Practice)
Security practices are established in the codebase. All features must include security considerations, input validation, and follow established security patterns.
"@
        }

        # Parse documentation data
        $docsParts = $scanResults.documentation -split '\|'
        $readmeCount = [int]$docsParts[0]

        if ($readmeCount -gt 0) {
            $additionalPrinciples += @"
### Documentation Matters (Project Practice)
Documentation practices are established with $readmeCount README files. All features must include appropriate documentation and maintain existing documentation standards.
"@
        }

        # Insert additional principles into constitution
        if ($additionalPrinciples.Count -gt 0) {
            $projectConstitution = $projectConstitution -replace '(## Additional Constraints)', @"
## Project-Specific Principles

$($additionalPrinciples -join "`n`n")

## Additional Constraints
"@
        }
    }

    # Inject custom constitution principles based on generic detection.
    $injectionResult = Inject-CustomPrinciples -Constitution $projectConstitution -RepoRoot $repoRoot -ScriptRoot $PSScriptRoot
    $projectConstitution = [string]$injectionResult.Constitution
    $injectedCustomPrinciples = @($injectionResult.InjectedPrincipleIds)

    # Validate inheritance
    if (-not $Json) {
        Test-ConstitutionInheritance -TeamConstitution $teamConstitution -ProjectConstitution $projectConstitution
        Write-Host ""
    }

    # Write constitution
    $projectConstitution | Out-File -FilePath $constitutionFile -Encoding UTF8

    # Output results
    $teamDirectivesPath = Get-TeamDirectivesPath
    if (-not $teamDirectivesPath) { $teamDirectivesPath = "" }
    if ($Json) {
        $result = @{
            status = "created"
            file = $constitutionFile
            team_directives = $teamDirectivesPath
            injected_custom_principles = $injectedCustomPrinciples
        } | ConvertTo-Json -Compress
        Write-Host $result
    } else {
        Write-Host "Constitution created at: $constitutionFile"
        Write-Host "Team directives source: $teamDirectivesPath"
        if ($injectedCustomPrinciples.Count -gt 0) {
            Write-Host "Injected custom principles: $($injectedCustomPrinciples -join ', ')"
        }
        Write-Host ""
        Write-Host "Next steps:"
        Write-Host "1. Review and customize the constitution for your project needs"
        Write-Host "2. Commit the constitution: git add .cx-spec/memory/constitution.md && git commit -m 'docs: initialize project constitution'"
        Write-Host "3. The constitution will be used by planning and implementation commands"
    }

} catch {
    Write-Error "Error: $_"
    exit 1
}
