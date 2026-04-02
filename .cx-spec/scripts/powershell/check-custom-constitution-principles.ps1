#!/usr/bin/env pwsh

[CmdletBinding()]
param(
    [switch]$Json,
    [switch]$Help
)

$ErrorActionPreference = 'Stop'

if ($Help) {
    Write-Output @"
Usage: check-custom-constitution-principles.ps1 [-Json] [-Help]

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
"@
    exit 0
}

. "$PSScriptRoot/common.ps1"

$repoRoot = Get-RepoRoot
$constitutionFile = Join-Path $repoRoot '.cx-spec/memory/constitution.md'

$activePrinciples = @()

if (Test-Path $constitutionFile -PathType Leaf) {
    $content = Get-Content -Path $constitutionFile -Raw -ErrorAction SilentlyContinue
    if ($null -eq $content) { $content = '' }

    $pattern = '(?m)^\s*#{2,6}\s+\[CP:([A-Za-z0-9_.-]+)\]\s+(.+?)\s*$'
    $matches = [regex]::Matches($content, $pattern, [System.Text.RegularExpressions.RegexOptions]::IgnoreCase)
    $sectionPattern = '(?m)^##\s+'

    $seen = @{}
    for ($i = 0; $i -lt $matches.Count; $i++) {
        $match = $matches[$i]
        $principleId = [string]$match.Groups[1].Value.Trim()
        if ([string]::IsNullOrWhiteSpace($principleId)) { continue }
        if ($seen.ContainsKey($principleId)) { continue }
        $seen[$principleId] = $true

        $start = $match.Index
        $end = if ($i + 1 -lt $matches.Count) { $matches[$i + 1].Index } else { $content.Length }
        $searchStart = $match.Index + $match.Length
        if ($searchStart -lt 0) { $searchStart = 0 }
        if ($searchStart -gt $content.Length) { $searchStart = $content.Length }
        $remainingText = $content.Substring($searchStart)
        $nextSectionMatch = [regex]::Match($remainingText, $sectionPattern)
        if ($nextSectionMatch.Success) {
            $sectionStart = $searchStart + $nextSectionMatch.Index
            if ($sectionStart -lt $end) {
                $end = $sectionStart
            }
        }
        $length = $end - $start
        if ($length -lt 0) { $length = 0 }
        $block = $content.Substring($start, $length).Trim()

        $title = [string]$match.Groups[2].Value.Trim()
        if ([string]::IsNullOrWhiteSpace($title)) { $title = $principleId }

        $activePrinciples += [ordered]@{
            id = $principleId
            title = $title
            content = $block
        }
    }
}

$payload = [ordered]@{
    active_principles = $activePrinciples
}

if ($Json) {
    ($payload | ConvertTo-Json -Depth 16 -Compress) | Write-Output
} else {
    if (@($activePrinciples).Count -gt 0) {
        Write-Output 'Custom constitution principles: loaded'
        foreach ($item in @($activePrinciples)) {
            Write-Output "- $($item.id): $($item.title)"
        }
    } else {
        Write-Output 'Custom constitution principles: none active'
    }
}

exit 0
