# PowerShell script to validate test grouping patterns
# Usage: .\scripts\validate-test-patterns.ps1

Write-Host "=== Integration Test Pattern Validation ===" -ForegroundColor Green
Write-Host ""

# Get all integration test functions
Write-Host "Extracting all integration test functions..." -ForegroundColor Yellow
$allTests = go test -tags integration -list . ./test/integration 2>&1 | Select-String "^Test"
$totalTests = ($allTests | Measure-Object).Count

Write-Host "Total integration tests found: $totalTests" -ForegroundColor Cyan
Write-Host ""

# Define test patterns for each group (same as in integration_up_parallel.sh)
$testGroups = @{
    "fast-validation" = "^Test(Auth|.*Configuration|Tenant|FeatureFlags|Predicate|.*Logs|FailProxyAuth)"
    "scan-core" = "^TestScan|^Test.*Scan.*"
    "scan-core-exclude" = "ASCA|Container|Realtime|Iac|Oss|Secrets|Kics|Scs"
    "scan-engines" = "^Test(.*ASCA|.*Asca|Container|Scs|.*Engine)"
    "scm-integration" = "^Test(PR|UserCount)"
    "realtime-features" = "^Test(Iac|Kics|Sca|Oss|Secrets|Containers)Realtime"
    "advanced-features" = "^Test(Project|Result|Import|.*Bfl|Chat|.*Learn|Telemetry|RateLimit|PreCommit|PreReceive|Remediation|GetProjectName)"
}

# Analyze each group
Write-Host "=== Test Distribution by Group ===" -ForegroundColor Green
Write-Host ""

$groupCounts = @{}
$totalMatched = 0
$testAssignments = @{}

foreach ($test in $allTests) {
    $testName = $test.ToString().Trim()
    $matched = $false
    $matchedGroups = @()
    
    foreach ($group in $testGroups.Keys) {
        if ($group -eq "scan-core-exclude") { continue }
        
        $pattern = $testGroups[$group]
        
        # Special handling for scan-core with exclusions
        if ($group -eq "scan-core") {
            if ($testName -match $pattern) {
                $excludePattern = $testGroups["scan-core-exclude"]
                if ($testName -notmatch $excludePattern) {
                    $matched = $true
                    $matchedGroups += $group
                }
            }
        }
        else {
            if ($testName -match $pattern) {
                $matched = $true
                $matchedGroups += $group
            }
        }
    }
    
    if ($matched) {
        $totalMatched++
        $testAssignments[$testName] = $matchedGroups
        
        foreach ($g in $matchedGroups) {
            if (-not $groupCounts.ContainsKey($g)) {
                $groupCounts[$g] = 0
            }
            $groupCounts[$g]++
        }
    }
}

# Display results
foreach ($group in @("fast-validation", "scan-core", "scan-engines", "scm-integration", "realtime-features", "advanced-features")) {
    $count = if ($groupCounts.ContainsKey($group)) { $groupCounts[$group] } else { 0 }
    Write-Host ("{0,-25}: {1,3} tests" -f $group, $count) -ForegroundColor Cyan
}

Write-Host ""
Write-Host "Total tests matched: $totalMatched" -ForegroundColor Cyan
$unmatched = $totalTests - $totalMatched
Write-Host "Unmatched tests: $unmatched" -ForegroundColor $(if ($unmatched -gt 0) { "Red" } else { "Green" })

# Show unmatched tests
if ($unmatched -gt 0) {
    Write-Host ""
    Write-Host "=== Unmatched Tests ===" -ForegroundColor Red
    foreach ($test in $allTests) {
        $testName = $test.ToString().Trim()
        if (-not $testAssignments.ContainsKey($testName)) {
            Write-Host "  - $testName" -ForegroundColor Yellow
        }
    }
}

# Check for duplicate matches
Write-Host ""
Write-Host "=== Checking for Duplicate Test Assignments ===" -ForegroundColor Green
Write-Host ""

$duplicatesFound = $false
foreach ($test in $testAssignments.Keys) {
    $groups = $testAssignments[$test]
    if ($groups.Count -gt 1) {
        Write-Host "⚠️  $test matches multiple groups: $($groups -join ', ')" -ForegroundColor Yellow
        $duplicatesFound = $true
    }
}

if (-not $duplicatesFound) {
    Write-Host "✅ No duplicate test assignments found" -ForegroundColor Green
}

# Summary
Write-Host ""
Write-Host "=== Summary ===" -ForegroundColor Green
Write-Host "Total tests: $totalTests" -ForegroundColor Cyan
Write-Host "Matched tests: $totalMatched" -ForegroundColor Cyan
Write-Host "Unmatched tests: $unmatched" -ForegroundColor $(if ($unmatched -gt 0) { "Red" } else { "Green" })
$coverage = [math]::Round(($totalMatched / $totalTests) * 100, 1)
Write-Host "Coverage: $coverage%" -ForegroundColor $(if ($coverage -eq 100) { "Green" } else { "Yellow" })

Write-Host ""
if ($unmatched -eq 0 -and -not $duplicatesFound) {
    Write-Host "✅ Test grouping is valid and complete!" -ForegroundColor Green
    exit 0
}
else {
    Write-Host "⚠️  Test grouping needs adjustment" -ForegroundColor Yellow
    exit 1
}
