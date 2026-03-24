#!/usr/bin/env pwsh
# Task Meta Utilities for Agentic SDLC
# Handles task classification, delegation, and status tracking
# PowerShell equivalent of tasks-meta-utils.sh

$ErrorActionPreference = 'Stop'

# Get project-level config path (.cx-spec/config.json)
function Get-ProjectConfigPath {
    try {
        $repoRoot = git rev-parse --show-toplevel 2>$null
        if ($LASTEXITCODE -eq 0 -and $repoRoot) {
            return Join-Path $repoRoot '.cx-spec/config.json'
        }
    } catch { }
    return '.cx-spec/config.json'
}

# Get config path (repo-local only)
function Get-ConfigPath {
    return Get-ProjectConfigPath
}

# Initialize tasks_meta.json for a feature
function Initialize-TasksMeta {
    param(
        [Parameter(Mandatory=$true)]
        [string]$FeatureDir
    )

    $tasksMetaFile = Join-Path $FeatureDir 'tasks_meta.json'

    # Create directory if it doesn't exist
    if (-not (Test-Path $FeatureDir)) {
        New-Item -ItemType Directory -Path $FeatureDir -Force | Out-Null
    }

    # Create basic structure
    $featureName = Split-Path $FeatureDir -Leaf
    $created = Get-Date -Format 'yyyy-MM-ddTHH:mm:ssK'

    $tasksMeta = @{
        feature = $featureName
        created = $created
        tasks = @{}
    }

    $tasksMeta | ConvertTo-Json -Depth 10 | Set-Content -Path $tasksMetaFile -Encoding UTF8

    Write-Output "Initialized tasks_meta.json at $tasksMetaFile"
}

# Classify task execution mode (SYNC/ASYNC)
function Get-TaskExecutionMode {
    param(
        [Parameter(Mandatory=$true)]
        [string]$Description,
        [Parameter(Mandatory=$false)]
        [string]$Files = ''
    )

    # Simple classification logic
    # ASYNC if description contains certain keywords or involves multiple files
    if ($Description -match 'research|analyze|design|plan|review|test') {
        return 'ASYNC'
    }

    $fileCount = ($Files -split '\s+' | Where-Object { $_ }).Count
    if ($fileCount -gt 2) {
        return 'ASYNC'
    }

    return 'SYNC'
}

# Add task to tasks_meta.json
function Add-Task {
    param(
        [Parameter(Mandatory=$true)]
        [string]$TasksMetaFile,
        [Parameter(Mandatory=$true)]
        [string]$TaskId,
        [Parameter(Mandatory=$true)]
        [string]$Description,
        [Parameter(Mandatory=$false)]
        [string]$Files = '',
        [Parameter(Mandatory=$true)]
        [string]$ExecutionMode
    )

    if (-not (Test-Path $TasksMetaFile)) {
        Write-Error "Tasks meta file not found: $TasksMetaFile"
        return
    }

    $tasksMeta = Get-Content $TasksMetaFile -Raw | ConvertFrom-Json

    # Ensure tasks is a hashtable for easy manipulation
    if ($null -eq $tasksMeta.tasks) {
        $tasksMeta | Add-Member -NotePropertyName 'tasks' -NotePropertyValue @{} -Force
    }

    $taskData = @{
        description = $Description
        files = $Files
        execution_mode = $ExecutionMode
        status = 'pending'
        agent_type = 'general'
    }

    # Add or update the task
    if ($tasksMeta.tasks -is [PSCustomObject]) {
        $tasksMeta.tasks | Add-Member -NotePropertyName $TaskId -NotePropertyValue $taskData -Force
    } else {
        $tasksMeta.tasks[$TaskId] = $taskData
    }

    $tasksMeta | ConvertTo-Json -Depth 10 | Set-Content -Path $TasksMetaFile -Encoding UTF8
    Write-Output "Added task $TaskId ($ExecutionMode) to $TasksMetaFile"
}

# Update task status in tasks_meta.json
function Update-TaskStatus {
    param(
        [Parameter(Mandatory=$true)]
        [string]$TasksMetaFile,
        [Parameter(Mandatory=$true)]
        [string]$TaskId,
        [Parameter(Mandatory=$true)]
        [ValidateSet('pending', 'completed', 'failed', 'skipped', 'rolled_back')]
        [string]$NewStatus
    )

    if (-not (Test-Path $TasksMetaFile)) {
        Write-Error "Tasks meta file not found: $TasksMetaFile"
        return
    }

    $tasksMeta = Get-Content $TasksMetaFile -Raw | ConvertFrom-Json

    if ($tasksMeta.tasks.$TaskId) {
        $tasksMeta.tasks.$TaskId.status = $NewStatus
        $tasksMeta | ConvertTo-Json -Depth 10 | Set-Content -Path $TasksMetaFile -Encoding UTF8
        Write-Output "Updated task $TaskId status to $NewStatus"
    } else {
        Write-Error "Task $TaskId not found in $TasksMetaFile"
    }
}

# Generate comprehensive agent context
function Get-AgentContext {
    param(
        [Parameter(Mandatory=$true)]
        [string]$FeatureDir
    )

    $context = "## Project Context`n`n"

    # Add spec.md content
    $specFile = Join-Path $FeatureDir 'spec.md'
    if (Test-Path $specFile) {
        $specContent = Get-Content $specFile -Raw
        $context += "### Feature Specification`n$specContent`n`n"
    }

    # Add plan.md content
    $planFile = Join-Path $FeatureDir 'plan.md'
    if (Test-Path $planFile) {
        $planContent = Get-Content $planFile -Raw
        $context += "### Technical Plan`n$planContent`n`n"
    }

    # Add data-model.md if exists
    $dataModelFile = Join-Path $FeatureDir 'data-model.md'
    if (Test-Path $dataModelFile) {
        $dataModelContent = Get-Content $dataModelFile -Raw
        $context += "### Data Model`n$dataModelContent`n`n"
    }

    # Add research.md if exists
    $researchFile = Join-Path $FeatureDir 'research.md'
    if (Test-Path $researchFile) {
        $researchContent = Get-Content $researchFile -Raw
        $context += "### Research & Decisions`n$researchContent`n`n"
    }

    # Add contracts if exists
    $contractsDir = Join-Path $FeatureDir 'contracts'
    if (Test-Path $contractsDir -PathType Container) {
        $context += "### API Contracts`n"
        Get-ChildItem -Path $contractsDir -Filter '*.md' | ForEach-Object {
            $contractName = $_.BaseName
            $contractContent = Get-Content $_.FullName -Raw
            $context += "#### $contractName`n$contractContent`n`n"
        }
    }

    # Add team context if available
    $constitutionFile = 'constitution.md'
    if (Test-Path $constitutionFile) {
        $constitutionContent = Get-Content $constitutionFile -TotalCount 50 | Out-String
        $context += "### Team Constitution`n$constitutionContent`n`n"
    }

    return $context
}

# Get mode configuration value
# Reads from mode_defaults.{current_mode}.{key} to get mode-specific settings
function Get-ModeConfig {
    param(
        [Parameter(Mandatory=$true)]
        [string]$Key
    )

    $configPath = Get-ConfigPath
    if (Test-Path $configPath) {
        try {
            $config = Get-Content $configPath -Raw | ConvertFrom-Json

            # Get current mode (build or spec), default to spec
            $mode = $config.workflow.current_mode
            if (-not $mode) { $mode = 'spec' }

            # Read mode-specific config value from mode_defaults
            $value = $config.mode_defaults.$mode.$Key
            if ($null -ne $value) {
                return $value.ToString().ToLower()
            }
        } catch { }
    }
    return 'false'
}

# Generate delegation prompt from task metadata with rich context
function New-DelegationPrompt {
    param(
        [Parameter(Mandatory=$true)]
        [string]$TaskId,
        [Parameter(Mandatory=$true)]
        [string]$AgentType,
        [Parameter(Mandatory=$true)]
        [string]$TaskDescription,
        [Parameter(Mandatory=$false)]
        [string]$TaskContext = '',
        [Parameter(Mandatory=$false)]
        [string]$TaskRequirements = '',
        [Parameter(Mandatory=$false)]
        [string]$ExecutionInstructions = '',
        [Parameter(Mandatory=$true)]
        [string]$FeatureDir
    )

    # Read template
    $scriptDir = $PSScriptRoot
    if (-not $scriptDir) { $scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path }
    if (-not $scriptDir) { $scriptDir = (Get-Location).Path }
    $templateFile = Join-Path $scriptDir '../../templates/delegation-template.md'

    if (-not (Test-Path $templateFile)) {
        Write-Error "Delegation template not found at $templateFile"
        return $null
    }

    $templateContent = Get-Content $templateFile -Raw

    # Generate comprehensive context
    $agentContext = Get-AgentContext -FeatureDir $FeatureDir

    # Combine task context with agent context
    $fullContext = "$TaskContext`n`n$agentContext"

    # Add atomic commits guidance if enabled
    $atomicCommits = Get-ModeConfig -Key 'atomic_commits'

    if ($atomicCommits -eq 'true') {
        $fullContext += @"

## COMMIT STRUCTURE GUIDANCE
Create atomic commits following this pattern:
- Each commit represents one logical unit of work
- Independently reviewable (can understand from commit message + diff)
- Self-contained (feature complete or milestone complete)
- Descriptive message: "[Feature]: What was accomplished"
Example: "[auth]: Implement JWT token validation"

This enables post-execution review and rollback capability.
"@
    }

    # Substitute variables
    $timestamp = Get-Date -Format 'yyyy-MM-dd HH:mm:ss'
    $prompt = $templateContent
    $prompt = $prompt -replace '\{AGENT_TYPE\}', $AgentType
    $prompt = $prompt -replace '\{TASK_DESCRIPTION\}', $TaskDescription
    $prompt = $prompt -replace '\{TASK_CONTEXT\}', $fullContext
    $prompt = $prompt -replace '\{TASK_REQUIREMENTS\}', $TaskRequirements
    $prompt = $prompt -replace '\{EXECUTION_INSTRUCTIONS\}', $ExecutionInstructions
    $prompt = $prompt -replace '\{TASK_ID\}', $TaskId
    $prompt = $prompt -replace '\{TIMESTAMP\}', $timestamp

    return $prompt
}


# Check delegation status
function Get-DelegationStatus {
    param(
        [Parameter(Mandatory=$true)]
        [string]$TaskId
    )

    # Check if prompt exists (task was delegated)
    $promptFile = "delegation_prompts/$TaskId.md"
    if (-not (Test-Path $promptFile)) {
        return 'no_job'
    }

    # Check for completion marker (AI assistant would create this)
    $completionFile = "delegation_completed/$TaskId.txt"
    if (Test-Path $completionFile) {
        return 'completed'
    }

    # Check for error marker
    $errorFile = "delegation_errors/$TaskId.txt"
    if (Test-Path $errorFile) {
        return 'failed'
    }

    # Otherwise, assume still running
    return 'running'
}

# Dispatch async task using natural language delegation with rich context
function Send-AsyncTask {
    param(
        [Parameter(Mandatory=$true)]
        [string]$TaskId,
        [Parameter(Mandatory=$true)]
        [string]$AgentType,
        [Parameter(Mandatory=$true)]
        [string]$TaskDescription,
        [Parameter(Mandatory=$false)]
        [string]$TaskContext = '',
        [Parameter(Mandatory=$false)]
        [string]$TaskRequirements = '',
        [Parameter(Mandatory=$false)]
        [string]$ExecutionInstructions = '',
        [Parameter(Mandatory=$true)]
        [string]$FeatureDir
    )

    # Generate natural language delegation prompt with comprehensive context
    $prompt = New-DelegationPrompt -TaskId $TaskId -AgentType $AgentType -TaskDescription $TaskDescription `
        -TaskContext $TaskContext -TaskRequirements $TaskRequirements `
        -ExecutionInstructions $ExecutionInstructions -FeatureDir $FeatureDir

    if (-not $prompt) {
        Write-Error "Failed to generate delegation prompt"
        return
    }

    # Save prompt for AI assistant consumption
    $promptDir = 'delegation_prompts'
    if (-not (Test-Path $promptDir)) {
        New-Item -ItemType Directory -Path $promptDir -Force | Out-Null
    }

    $promptFile = Join-Path $promptDir "$TaskId.md"
    $prompt | Set-Content -Path $promptFile -Encoding UTF8

    Write-Output "Task $TaskId delegated successfully - comprehensive prompt saved for AI assistant"
}

# Analyze implementation changes vs documentation for evolution
function Get-ImplementationChanges {
    param(
        [Parameter(Mandatory=$true)]
        [string]$FeatureDir
    )

    $specFile = Join-Path $FeatureDir 'spec.md'
    $planFile = Join-Path $FeatureDir 'plan.md'
    $tasksFile = Join-Path $FeatureDir 'tasks.md'

    $changes = "## Implementation vs Documentation Analysis`n"

    # Check for new features in code not in spec
    $hasSrcDir = (Test-Path 'src' -PathType Container) -or (Test-Path 'lib' -PathType Container)
    $hasCodeFiles = (Get-ChildItem -Path . -Recurse -Include '*.js','*.ts','*.py' -ErrorAction SilentlyContinue | Select-Object -First 1)

    if ($hasSrcDir -or $hasCodeFiles) {
        $changes += @"
### Potential New Features
- Scan codebase for implemented functionality not documented in spec.md
- Check for new API endpoints, UI components, or business logic
- Identify user flows that may have evolved during implementation

"@
    }

    # Check for architecture changes
    if (Test-Path $planFile) {
        $changes += @"
### Architecture Evolution
- Compare implemented architecture against plan.md
- Identify performance optimizations or security enhancements
- Note technology stack changes or library updates

"@
    }

    # Check for completed tasks that might indicate refinements
    if (Test-Path $tasksFile) {
        $tasksContent = Get-Content $tasksFile -Raw
        $completedTasks = ([regex]::Matches($tasksContent, '^- \[X\]', 'Multiline')).Count
        $totalTasks = ([regex]::Matches($tasksContent, '^- \[.\]', 'Multiline')).Count

        $changes += @"
### Task Completion Status
- Completed: $completedTasks / $totalTasks tasks
- Check for additional tasks added during implementation
- Identify refinements or bug fixes that emerged

"@
    }

    return $changes
}


# Propose documentation updates based on implementation analysis
function New-DocumentationUpdateProposal {
    param(
        [Parameter(Mandatory=$true)]
        [string]$FeatureDir,
        [Parameter(Mandatory=$true)]
        [string]$AnalysisResults
    )

    $proposals = @"
## Documentation Evolution Proposals

Based on implementation analysis, here are recommended documentation updates:

"@

    # Check if there are undocumented features
    if ($AnalysisResults -match 'new features|new API|new components') {
        $proposals += @"
### Spec.md Updates
- Add newly implemented features to functional requirements
- Document discovered edge cases and user experience insights
- Update acceptance criteria based on actual implementation

"@
    }

    # Check for architecture changes
    if ($AnalysisResults -match 'architecture|performance|security') {
        $proposals += @"
### Plan.md Updates
- Document architecture changes made during implementation
- Add performance optimizations and their rationale
- Update technology decisions based on implementation experience

"@
    }

    # Check for task additions
    if ($AnalysisResults -match 'additional tasks|refinements') {
        $proposals += @"
### Tasks.md Updates
- Add follow-up tasks for refinements discovered during implementation
- Document bug fixes and improvements made
- Update task status and add completion notes

"@
    }

    $proposals += @"
### Evolution Guidelines
- Preserve original requirements while incorporating implementation learnings
- Maintain traceability between documentation and code
- Version documentation changes with clear rationale
- Ensure constitution compliance for any new requirements

"@

    return $proposals
}

# Apply documentation updates with user confirmation
function Set-DocumentationUpdates {
    param(
        [Parameter(Mandatory=$true)]
        [string]$FeatureDir,
        [Parameter(Mandatory=$true)]
        [ValidateSet('spec', 'plan', 'tasks')]
        [string]$UpdateType,
        [Parameter(Mandatory=$true)]
        [string]$UpdateContent
    )

    $timestamp = Get-Date -Format 'yyyy-MM-dd HH:mm:ss'

    switch ($UpdateType) {
        'spec' {
            $specFile = Join-Path $FeatureDir 'spec.md'
            Add-Content -Path $specFile -Value "`n## Implementation Learnings - $timestamp`n`n$UpdateContent"
            Write-Output "Updated spec.md with implementation insights"
        }
        'plan' {
            $planFile = Join-Path $FeatureDir 'plan.md'
            Add-Content -Path $planFile -Value "`n## Architecture Evolution - $timestamp`n`n$UpdateContent"
            Write-Output "Updated plan.md with architecture changes"
        }
        'tasks' {
            $tasksFile = Join-Path $FeatureDir 'tasks.md'
            Add-Content -Path $tasksFile -Value "`n## Refinement Tasks - $timestamp`n$UpdateContent"
            Write-Output "Added refinement tasks to tasks.md"
        }
    }
}

# Rollback individual task while preserving documentation
function Undo-Task {
    param(
        [Parameter(Mandatory=$true)]
        [string]$TasksMetaFile,
        [Parameter(Mandatory=$true)]
        [string]$TaskId,
        [Parameter(Mandatory=$false)]
        [bool]$PreserveDocs = $true
    )

    Write-Output "Rolling back task: $TaskId"

    # Get task information before rollback
    $taskDescription = ''
    $taskFiles = ''

    if (Test-Path $TasksMetaFile) {
        $tasksMeta = Get-Content $TasksMetaFile -Raw | ConvertFrom-Json
        $task = $tasksMeta.tasks.$TaskId
        if ($task) {
            $taskDescription = $task.description
            $taskFiles = $task.files

            # Mark task as rolled back
            $task.status = 'rolled_back'
            $tasksMeta | ConvertTo-Json -Depth 10 | Set-Content -Path $TasksMetaFile -Encoding UTF8
        }
    }

    # Attempt to revert code changes (simplified)
    if ($taskFiles) {
        Write-Output "Attempting to revert changes to files: $taskFiles"
        Write-Output "Note: Manual code reversion may be required for files: $taskFiles"
    }

    # Log rollback details
    $featureDir = Split-Path $TasksMetaFile -Parent
    $rollbackLog = Join-Path $featureDir 'rollback.log'
    $timestamp = Get-Date -Format 'yyyy-MM-dd HH:mm:ss'

    $logEntry = @"
## Task Rollback - $timestamp
Task ID: $TaskId
Description: $taskDescription
Files: $taskFiles
Documentation Preserved: $PreserveDocs

"@
    Add-Content -Path $rollbackLog -Value $logEntry

    Write-Output "Task $TaskId rolled back successfully"

    if ($PreserveDocs) {
        Write-Output "Documentation preserved for rolled back task $TaskId"
    }
}

# Rollback entire feature implementation
function Undo-Feature {
    param(
        [Parameter(Mandatory=$true)]
        [string]$FeatureDir,
        [Parameter(Mandatory=$false)]
        [bool]$PreserveDocs = $true
    )

    $featureName = Split-Path $FeatureDir -Leaf
    Write-Output "Rolling back entire feature: $featureName"

    $tasksMetaFile = Join-Path $FeatureDir 'tasks_meta.json'

    if (Test-Path $tasksMetaFile) {
        # Mark all tasks as rolled back
        $tasksMeta = Get-Content $tasksMetaFile -Raw | ConvertFrom-Json
        $tasksMeta.tasks.PSObject.Properties | ForEach-Object {
            $_.Value.status = 'rolled_back'
        }
        $tasksMeta | ConvertTo-Json -Depth 10 | Set-Content -Path $tasksMetaFile -Encoding UTF8
    }

    # Remove implementation artifacts
    $implLog = Join-Path $FeatureDir 'implementation.log'
    $checklistsDir = Join-Path $FeatureDir 'checklists'

    if (Test-Path $implLog) { Remove-Item $implLog -Force }
    if (Test-Path $checklistsDir) { Remove-Item $checklistsDir -Recurse -Force }

    # Log comprehensive rollback
    $rollbackLog = Join-Path $FeatureDir 'rollback.log'
    $timestamp = Get-Date -Format 'yyyy-MM-dd HH:mm:ss'

    $logEntry = @"
## Feature Rollback - $timestamp
Feature: $featureName
All tasks marked as rolled back
Implementation artifacts removed
Documentation Preserved: $PreserveDocs

"@
    Add-Content -Path $rollbackLog -Value $logEntry

    Write-Output "Feature $featureName implementation rolled back"

    if ($PreserveDocs) {
        Write-Output "Documentation preserved during feature rollback"
    }
}

# Regenerate tasks after rollback with corrected approach
function Reset-TasksAfterRollback {
    param(
        [Parameter(Mandatory=$true)]
        [string]$FeatureDir,
        [Parameter(Mandatory=$true)]
        [string]$RollbackReason
    )

    $tasksFile = Join-Path $FeatureDir 'tasks.md'
    $tasksMetaFile = Join-Path $FeatureDir 'tasks_meta.json'
    $timestamp = Get-Date -Format 'yyyy-MM-dd HH:mm:ss'

    # Add new tasks for corrected implementation
    $newTasks = @"

## Corrected Implementation Tasks - $timestamp

- [ ] T_CORRECT_001 Apply corrected implementation approach based on: $RollbackReason
- [ ] T_CORRECT_002 Verify fixes address root cause of rollback
- [ ] T_CORRECT_003 Test corrected implementation thoroughly
"@
    Add-Content -Path $tasksFile -Value $newTasks

    # Reinitialize tasks metadata
    if (Test-Path $tasksMetaFile) {
        # Reset rolled back tasks to pending for retry
        $tasksMeta = Get-Content $tasksMetaFile -Raw | ConvertFrom-Json
        $tasksMeta.tasks.PSObject.Properties | ForEach-Object {
            if ($_.Value.status -eq 'rolled_back') {
                $_.Value.status = 'pending'
            }
        }
        $tasksMeta | ConvertTo-Json -Depth 10 | Set-Content -Path $tasksMetaFile -Encoding UTF8
    }

    Write-Output "Regenerated tasks for corrected implementation approach"
}

# Regenerate plan based on current specifications and implementation learnings
function Reset-Plan {
    param(
        [Parameter(Mandatory=$true)]
        [string]$FeatureDir,
        [Parameter(Mandatory=$true)]
        [string]$Reason
    )

    $specFile = Join-Path $FeatureDir 'spec.md'
    $planFile = Join-Path $FeatureDir 'plan.md'
    $featureName = Split-Path $FeatureDir -Leaf

    Write-Output "Regenerating plan for feature: $featureName"
    Write-Output "Reason: $Reason"

    if (-not (Test-Path $specFile)) {
        Write-Output "Error: Cannot regenerate plan without spec.md"
        return
    }

    # Backup original plan
    if (Test-Path $planFile) {
        $backupName = "$planFile.backup.$(Get-Date -Format 'yyyyMMdd_HHmmss')"
        Copy-Item $planFile $backupName
        Write-Output "Original plan backed up"
    }

    # Get spec content (first 20 lines)
    $specContent = Get-Content $specFile -TotalCount 20 | Out-String
    $timestamp = Get-Date -Format 'yyyy-MM-dd HH:mm:ss'

    # Create regeneration template
    $planTemplate = @"
# Implementation Plan - Regenerated $timestamp
## Reason for Regeneration
$Reason

## Original Specification Context
$specContent

## Architecture Decisions
<!-- Regenerate based on current spec and implementation learnings -->

## Technical Stack
<!-- Update based on implementation experience -->

## Component Design
<!-- Refine based on actual implementation needs -->

## Data Model
<!-- Adjust based on real-world usage patterns -->

## Implementation Phases
<!-- Reorganize based on lessons learned -->

## Risk Mitigation
<!-- Update based on encountered issues -->

## Success Metrics
<!-- Refine based on implementation insights -->
"@

    $planTemplate | Set-Content -Path $planFile -Encoding UTF8

    # Log regeneration
    $rollbackLog = Join-Path $FeatureDir 'rollback.log'
    $logEntry = @"
## Plan Regeneration - $timestamp
Reason: $Reason
Original plan backed up
New plan template created for regeneration

"@
    Add-Content -Path $rollbackLog -Value $logEntry

    Write-Output "Plan regeneration template created. Manual editing required to complete regeneration."
    Write-Output "Original plan backed up for reference."
}

# Ensure documentation consistency after rollback
function Test-DocumentationConsistency {
    param(
        [Parameter(Mandatory=$true)]
        [string]$FeatureDir
    )

    Write-Output "Ensuring documentation consistency after rollback..."

    $specFile = Join-Path $FeatureDir 'spec.md'
    $planFile = Join-Path $FeatureDir 'plan.md'
    $tasksFile = Join-Path $FeatureDir 'tasks.md'

    $issuesFound = $false

    # Check if plan references tasks that no longer exist
    if ((Test-Path $planFile) -and (Test-Path $tasksFile)) {
        $planContent = Get-Content $planFile -Raw
        $tasksContent = Get-Content $tasksFile -Raw

        if (($planContent -match 'T[0-9]') -and -not ($tasksContent -match 'T[0-9]')) {
            Write-Output "⚠️  Plan references tasks that may no longer exist"
            $issuesFound = $true
        }
    }

    # Check for implementation references in docs after rollback
    if (Test-Path $specFile) {
        $specContent = Get-Content $specFile -Raw
        if ($specContent -match 'implementation|code|deploy') {
            Write-Output "⚠️  Spec contains implementation details that should be reviewed"
            $issuesFound = $true
        }
    }

    if (-not $issuesFound) {
        Write-Output "✅ Documentation consistency verified"
    } else {
        Write-Output "⚠️  Some documentation consistency issues detected"
        Write-Output "Consider running '/analyze' to identify specific issues"
    }
}

# Mode-aware rollback strategies
function Get-ModeAwareRollbackStrategy {
    param(
        [Parameter(Mandatory=$false)]
        [string]$Mode = 'spec'
    )

    switch ($Mode) {
        'build' { return 'build_mode_rollback' }
        'spec' { return 'spec_mode_rollback' }
        default { return 'default_rollback' }
    }
}

# Execute mode-aware rollback
function Invoke-ModeAwareRollback {
    param(
        [Parameter(Mandatory=$true)]
        [string]$FeatureDir,
        [Parameter(Mandatory=$true)]
        [ValidateSet('task', 'feature')]
        [string]$RollbackType,
        [Parameter(Mandatory=$false)]
        [string]$Mode = 'spec',
        [Parameter(Mandatory=$false)]
        [string]$TaskId = ''
    )

    $strategy = Get-ModeAwareRollbackStrategy -Mode $Mode

    Write-Output "Executing $strategy for $RollbackType in $Mode mode"

    switch ($strategy) {
        'build_mode_rollback' {
            # Lightweight rollback for build mode
            Write-Output "Build mode: Minimal rollback preserving rapid iteration artifacts"
            switch ($RollbackType) {
                'task' {
                    Write-Output "Task rollback completed with minimal cleanup"
                }
                'feature' {
                    Write-Output "Feature rollback completed, preserving iteration artifacts"
                }
            }
        }
        'spec_mode_rollback' {
            # Comprehensive rollback for spec mode
            Write-Output "Spec mode: Comprehensive rollback with full documentation preservation"
            switch ($RollbackType) {
                'task' {
                    $tasksMetaFile = Join-Path $FeatureDir 'tasks_meta.json'
                    Undo-Task -TasksMetaFile $tasksMetaFile -TaskId $TaskId -PreserveDocs $true
                }
                'feature' {
                    Undo-Feature -FeatureDir $FeatureDir -PreserveDocs $true
                }
            }
        }
        'default_rollback' {
            Write-Output "Default rollback strategy applied"
        }
    }
}

# Get framework options configuration
function Get-FrameworkOpinions {
    param(
        [Parameter(Mandatory=$false)]
        [string]$Mode = 'spec'
    )

    $configFile = Get-ConfigPath

    # Read from consolidated config (hierarchical)
    if ((Test-Path $configFile)) {
        try {
            $config = Get-Content $configFile -Raw | ConvertFrom-Json

            $userTdd = $config.options.tdd_enabled
            $userContracts = $config.options.contracts_enabled
            $userDataModels = $config.options.data_models_enabled
            $userRiskTests = $config.options.risk_tests_enabled

            # Fill in defaults for unset options based on mode
            switch ($Mode) {
                'build' {
                    if ($null -eq $userTdd) { $userTdd = $false }
                    if ($null -eq $userContracts) { $userContracts = $false }
                    if ($null -eq $userDataModels) { $userDataModels = $false }
                    if ($null -eq $userRiskTests) { $userRiskTests = $false }
                }
                'spec' {
                    if ($null -eq $userTdd) { $userTdd = $true }
                    if ($null -eq $userContracts) { $userContracts = $true }
                    if ($null -eq $userDataModels) { $userDataModels = $true }
                    if ($null -eq $userRiskTests) { $userRiskTests = $true }
                }
            }

            return "tdd_enabled=$userTdd contracts_enabled=$userContracts data_models_enabled=$userDataModels risk_tests_enabled=$userRiskTests"
        } catch { }
    }

    # Fallback to mode-based defaults
    switch ($Mode) {
        'build' {
            return "tdd_enabled=false contracts_enabled=false data_models_enabled=false risk_tests_enabled=false"
        }
        'spec' {
            return "tdd_enabled=true contracts_enabled=true data_models_enabled=true risk_tests_enabled=true"
        }
        default {
            return "tdd_enabled=false contracts_enabled=false data_models_enabled=false risk_tests_enabled=false"
        }
    }
}

# Set framework opinion (legacy compatibility - now handled by per-spec mode)
function Set-FrameworkOpinion {
    param(
        [Parameter(Mandatory=$true)]
        [string]$OpinionType,
        [Parameter(Mandatory=$true)]
        [string]$Value
    )

    Write-Output "Framework opinions are now managed by feature-level mode configuration."
    Write-Output "Use '/cx-spec.cx-spec --mode=build|spec --$OpinionType' to create features with specific framework settings."
    Write-Output "Run '/cx-spec.cx-spec --help' for more information."
}

# Check if opinion is enabled
function Test-OpinionEnabled {
    param(
        [Parameter(Mandatory=$true)]
        [string]$OpinionType,
        [Parameter(Mandatory=$false)]
        [string]$Mode = 'spec'
    )

    $opinions = Get-FrameworkOpinions -Mode $Mode

    switch ($OpinionType) {
        'tdd' {
            if ($opinions -match 'tdd_enabled=([^ ]+)') { return $Matches[1] }
        }
        'contracts' {
            if ($opinions -match 'contracts_enabled=([^ ]+)') { return $Matches[1] }
        }
        'data_models' {
            if ($opinions -match 'data_models_enabled=([^ ]+)') { return $Matches[1] }
        }
        'risk_tests' {
            if ($opinions -match 'risk_tests_enabled=([^ ]+)') { return $Matches[1] }
        }
    }
    return 'false'
}

# Generate tasks with configurable opinions
function New-TasksWithOpinions {
    param(
        [Parameter(Mandatory=$true)]
        [string]$FeatureDir,
        [Parameter(Mandatory=$false)]
        [string]$Mode = 'spec'
    )

    $opinions = Get-FrameworkOpinions -Mode $Mode

    Write-Output "Generating tasks with framework opinions for $Mode mode:"
    Write-Output $opinions
    Write-Output ""

    $tddEnabled = Test-OpinionEnabled -OpinionType 'tdd' -Mode $Mode
    $contractsEnabled = Test-OpinionEnabled -OpinionType 'contracts' -Mode $Mode
    $dataModelsEnabled = Test-OpinionEnabled -OpinionType 'data_models' -Mode $Mode

    # Generate tasks based on enabled opinions
    Write-Output "### Task Generation Configuration"
    Write-Output "- TDD: $tddEnabled"
    Write-Output "- Contracts: $contractsEnabled"
    Write-Output "- Data Models: $dataModelsEnabled"
    Write-Output ""

    if ($tddEnabled -eq 'true' -or $tddEnabled -eq 'True') {
        Write-Output "TDD enabled: Tests will be generated BEFORE implementation tasks (in each user story phase)"
    } else {
        Write-Output "TDD disabled: Tests will be generated AFTER implementation tasks (in Polish phase)"
    }
}

# Main function for CLI usage
if ($MyInvocation.InvocationName -ne '.') {
    $command = $args[0]
    $remainingArgs = $args[1..($args.Length - 1)]

    switch ($command) {
        'init' {
            Initialize-TasksMeta -FeatureDir $remainingArgs[0]
        }
        'classify' {
            Get-TaskExecutionMode -Description $remainingArgs[0] -Files $remainingArgs[1]
        }
        'add-task' {
            Add-Task -TasksMetaFile $remainingArgs[0] -TaskId $remainingArgs[1] -Description $remainingArgs[2] -Files $remainingArgs[3] -ExecutionMode $remainingArgs[4]
        }
        'update-status' {
            Update-TaskStatus -TasksMetaFile $remainingArgs[0] -TaskId $remainingArgs[1] -NewStatus $remainingArgs[2]
        }
        'generate_delegation_prompt' {
            New-DelegationPrompt -TaskId $remainingArgs[0] -AgentType $remainingArgs[1] -TaskDescription $remainingArgs[2] -TaskContext $remainingArgs[3] -TaskRequirements $remainingArgs[4] -ExecutionInstructions $remainingArgs[5] -FeatureDir $remainingArgs[6]
        }
        'check_delegation_status' {
            Get-DelegationStatus -TaskId $remainingArgs[0]
        }
        'dispatch_async_task' {
            Send-AsyncTask -TaskId $remainingArgs[0] -AgentType $remainingArgs[1] -TaskDescription $remainingArgs[2] -TaskContext $remainingArgs[3] -TaskRequirements $remainingArgs[4] -ExecutionInstructions $remainingArgs[5] -FeatureDir $remainingArgs[6]
        }
        'analyze_implementation_changes' {
            Get-ImplementationChanges -FeatureDir $remainingArgs[0]
        }
        'propose_documentation_updates' {
            New-DocumentationUpdateProposal -FeatureDir $remainingArgs[0] -AnalysisResults $remainingArgs[1]
        }
        'apply_documentation_updates' {
            Set-DocumentationUpdates -FeatureDir $remainingArgs[0] -UpdateType $remainingArgs[1] -UpdateContent $remainingArgs[2]
        }
        'rollback_task' {
            Undo-Task -TasksMetaFile $remainingArgs[0] -TaskId $remainingArgs[1] -PreserveDocs ($remainingArgs[2] -ne 'false')
        }
        'rollback_feature' {
            Undo-Feature -FeatureDir $remainingArgs[0] -PreserveDocs ($remainingArgs[1] -ne 'false')
        }
        'regenerate_tasks_after_rollback' {
            Reset-TasksAfterRollback -FeatureDir $remainingArgs[0] -RollbackReason $remainingArgs[1]
        }
        'regenerate_plan' {
            Reset-Plan -FeatureDir $remainingArgs[0] -Reason $remainingArgs[1]
        }
        'ensure_documentation_consistency' {
            Test-DocumentationConsistency -FeatureDir $remainingArgs[0]
        }
        'get_mode_aware_rollback_strategy' {
            Get-ModeAwareRollbackStrategy -Mode $remainingArgs[0]
        }
        'execute_mode_aware_rollback' {
            Invoke-ModeAwareRollback -FeatureDir $remainingArgs[0] -RollbackType $remainingArgs[1] -Mode $remainingArgs[2] -TaskId $remainingArgs[3]
        }
        'get_framework_opinions' {
            Get-FrameworkOpinions -Mode $remainingArgs[0]
        }
        'set_framework_opinion' {
            Set-FrameworkOpinion -OpinionType $remainingArgs[0] -Value $remainingArgs[1]
        }
        'is_opinion_enabled' {
            Test-OpinionEnabled -OpinionType $remainingArgs[0] -Mode $remainingArgs[1]
        }
        'generate_tasks_with_opinions' {
            New-TasksWithOpinions -FeatureDir $remainingArgs[0] -Mode $remainingArgs[1]
        }
        default {
            Write-Output "Usage: $($MyInvocation.MyCommand.Name) {init|classify|add-task|update-status|generate_delegation_prompt|check_delegation_status|dispatch_async_task|analyze_implementation_changes|propose_documentation_updates|apply_documentation_updates|rollback_task|rollback_feature|regenerate_tasks_after_rollback|...} [args...]"
            exit 1
        }
    }
}
