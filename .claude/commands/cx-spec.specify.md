---
description: Create or update feature spec from description
handoffs:
  - label: Build Technical Plan
    agent: cx-spec.plan
    prompt: Create a plan for the spec. I am building with...
  - label: Clarify Spec Requirements
    agent: cx-spec.clarify
    prompt: Clarify specification requirements
    send: true
---

## User Input

```text
$ARGUMENTS
```


You **MUST** consider the user input before proceeding (if not empty).

## Shell Compatibility

- Use `.cx-spec/scripts/bash/...` on macOS/Linux.
- Use `.cx-spec/scripts/powershell/...` on Windows.
- If a step shows only one shell form, use the equivalent script in the other shell.





## Parameters

Parse the following parameters from `$ARGUMENTS`:

- `--mode=build|spec`: Workflow mode (default: spec)
- `--tdd`: Enable TDD (overrides mode default)
- `--no-tdd`: Disable TDD (overrides mode default)
- `--contracts`: Enable API contracts (overrides mode default)
- `--no-contracts`: Disable API contracts (overrides mode default)
- `--data-models`: Enable data models (overrides mode default)
- `--no-data-models`: Disable data models (overrides mode default)
- `--risk-tests`: Enable risk-based testing (overrides mode default)
- `--no-risk-tests`: Disable risk-based testing (overrides mode default)
- `--architecture`: Enable feature-level architecture generation during planning
- `--no-architecture`: Disable feature-level architecture generation

**Mode-Specific Defaults**:

- **Build Mode**: tdd=false, contracts=false, data_models=false, risk_tests=false, architecture=true
- **Spec Mode**: tdd=false, contracts=true, data_models=true, risk_tests=true, architecture=true

**Note**: Architecture is enabled by default in both modes. Use `--no-architecture` to disable feature-level `specs/{feature}/architecture.md` during `/cx-spec.plan`.

After parsing, extract the feature description (everything after parameters).

## Mode & Options Resolution

1. **Determine Effective Mode**: Parse `--mode` from arguments, default to "spec" if not specified

2. **Determine Effective Options**:
   - Start with mode-specific defaults
   - Override with explicit flags (e.g., `--no-tdd` overrides default)
   - Pass to script as: `--mode build --tdd false --contracts false --data-models false --risk-tests false`

3. **Mode-Aware Behavior**:
   - **Build Mode**: Lightweight, conversational specification focused on quick validation and exploration
   - **Spec Mode**: Full structured specification with comprehensive requirements and validation

## Outline

The text the user typed after `/cx-spec.specify` in the triggering message **is** the feature description. Assume you always have it available in this conversation even if `$ARGUMENTS` appears literally below. Do not ask the user to repeat it unless they provided an empty command.

Given that feature description, do this:

1. **Require a JIRA ID in the feature description**:
   - The feature description must include a ticket key like `SCA-123456`.
   - Use the format: `<JIRA-ID> <feature description>`.
   - Example: `SCA-123456 add login with email and password`.

2. **Generate a concise short name** (2-4 words) for the branch suffix:
   - Analyze the feature description and extract the most meaningful keywords
   - Create a 2-4 word short name that captures the essence of the feature
   - Use action-noun format when possible (e.g., "add-user-auth", "fix-payment-bug")
   - Preserve technical terms and acronyms (OAuth2, API, JWT, etc.)
   - Keep it concise but descriptive enough to understand the feature at a glance
   - Examples:
     - "SCA-123456 I want to add user authentication" → `sca-123456-user-auth`
     - "SCA-123456 Implement OAuth2 integration for the API" → `sca-123456-oauth2-api-integration`
     - "SCA-123456 Create a dashboard for analytics" → `sca-123456-analytics-dashboard`

3. **Run create-new-feature exactly once** with mode/options and the original description:
   - Bash example:
     ```bash
     .cx-spec/scripts/bash/create-new-feature.sh --json --short-name "user-auth" --mode spec --tdd true --contracts true --data-models true --risk-tests true "SCA-123456 Add user authentication"
     ```
   - PowerShell example:
     ```powershell
     .cx-spec/scripts/powershell/create-new-feature.ps1 -Json -ShortName "user-auth" -Mode spec -Tdd -Contracts -DataModels -RiskTests "SCA-123456 Add user authentication"
     ```

   **IMPORTANT**:
   - Do not compute numeric branch IDs or pass `--number`; JIRA naming is the source of truth.
   - The generated branch/spec directory format is `<jira-lowercase>-<short-name>` (example: `sca-123456-add-login`).
   - You must only run the create script once per feature.
   - Pass mode and all four options (tdd, contracts, data_models, risk_tests).
   - The JSON output contains `BRANCH_NAME` and `SPEC_FILE`; always use those values.
   - For single quotes in args like "I'm Groot", use escape syntax: e.g 'I'\''m Groot' (or double-quote if possible: "I'm Groot")

   **SCRIPT AUTO-CREATES FILES** (do NOT manually read templates or create these files):
   - The script automatically copies the appropriate template to `SPEC_FILE` (spec.md)
   - The script automatically creates `context.md` with intelligent defaults based on the feature description
   - After the script runs, **read and edit the already-created files** - do not re-read templates

4. Follow this execution flow (mode-aware):

     **Build Mode Execution Flow:**
     1. Parse user description from Input
        If empty: ERROR "No feature description provided"
     2. Extract key concepts from description
        Identify: actors, actions, data, constraints
     3. For unclear aspects:
        - Make informed guesses based on context and industry standards
        - Only mark with [NEEDS CLARIFICATION: specific question] if critical for basic functionality
        - **LIMIT: Maximum 1 [NEEDS CLARIFICATION] marker total** (keep it minimal)
        - Focus only on scope-defining decisions
     4. Fill User Scenarios & Testing section (lightweight)
        - Focus on 1-2 primary user journeys
        - Simple acceptance scenarios (Given/When/Then format)
        - If no clear user flow: ERROR "Cannot determine user scenarios"
     5. Generate Functional Requirements (simplified)
        - Focus on core functionality only
        - Use reasonable defaults for unspecified details
        - Keep to 3-5 key requirements
     6. Define Success Criteria (basic)
        - 2-3 measurable outcomes focused on core functionality
        - Technology-agnostic but practical
     7. Identify Key Entities (if data involved, minimal)
        - Only essential entities and relationships
     8. Return: SUCCESS (spec ready for lightweight implementation)

     **Spec Mode Execution Flow:**
     1. Parse user description from Input
        If empty: ERROR "No feature description provided"
     2. Extract key concepts from description
        Identify: actors, actions, data, constraints
     3. For unclear aspects:
        - Make informed guesses based on context and industry standards
        - Only mark with [NEEDS CLARIFICATION: specific question] if:
          - The choice significantly impacts feature scope or user experience
          - Multiple reasonable interpretations exist with different implications
          - No reasonable default exists
        - **LIMIT: Maximum 3 [NEEDS CLARIFICATION] markers total**
        - Prioritize clarifications by impact: scope > security/privacy > user experience > technical details
     4. Fill User Scenarios & Testing section
        If no clear user flow: ERROR "Cannot determine user scenarios"
     5. Generate Functional Requirements
        Each requirement must be testable
        Use reasonable defaults for unspecified details (document assumptions in Assumptions section)
     6. Define Success Criteria
        Create measurable, technology-agnostic outcomes
        Include both quantitative metrics (time, performance, volume) and qualitative measures (user satisfaction, task completion)
        Each criterion must be verifiable without implementation details
     7. Identify Key Entities (if data involved)
     8. Return: SUCCESS (spec ready for planning)

6. Write the specification to SPEC_FILE using the template structure, replacing placeholders with concrete details derived from the feature description (arguments) while preserving section order and headings.

7. **Specification Quality Validation** (mode-aware):

    **Build Mode Validation:**
    - **Lightweight Checklist**: Focus on core functionality and basic testability
    - **Reduced Requirements**: Skip detailed edge cases and comprehensive coverage
    - **Quick Validation**: 1-2 iteration maximum, prioritize getting something working
    - **Success Criteria**: Basic functionality demonstrable, core user journey works

      - [ ] No implementation details (languages, frameworks, APIs)
      - [ ] Focused on user value and business needs
      - [ ] Written for non-technical stakeholders
      - [ ] All mandatory sections completed

      ## Requirement Completeness

      - [ ] No [NEEDS CLARIFICATION] markers remain
      - [ ] Requirements are testable and unambiguous
      - [ ] Success criteria are measurable
      - [ ] Success criteria are technology-agnostic (no implementation details)
      - [ ] All acceptance scenarios are defined
      - [ ] Edge cases are identified
      - [ ] Scope is clearly bounded
      - [ ] Dependencies and assumptions identified

      ## Feature Readiness

      - [ ] All functional requirements have clear acceptance criteria
      - [ ] User scenarios cover primary flows
      - [ ] Feature meets measurable outcomes defined in Success Criteria
      - [ ] No implementation details leak into specification

      ## Notes

       - Items marked incomplete require spec updates before `/cx-spec.clarify` or `/cx-spec.plan`

   b. **Run Validation Check**: Review the spec against each checklist item:
      - For each item, determine if it passes or fails
      - Document specific issues found (quote relevant spec sections)

   c. **Handle Validation Results**:

      - **If all items pass**: Mark checklist complete and proceed to the next step

      - **If items fail (excluding [NEEDS CLARIFICATION])**:
        1. List the failing items and specific issues
        2. Update the spec to address each issue
        3. Re-run validation until all items pass (max 3 iterations)
        4. If still failing after 3 iterations, document remaining issues in checklist notes and warn user

      - **If [NEEDS CLARIFICATION] markers remain**:
        1. Extract all [NEEDS CLARIFICATION: ...] markers from the spec
        2. **LIMIT CHECK**: If more than 3 markers exist, keep only the 3 most critical (by scope/security/UX impact) and make informed guesses for the rest
        3. For each clarification needed (max 3), present options to user in this format:

           ```markdown
           ## Question [N]: [Topic]
           
           **Context**: [Quote relevant spec section]
           
           **What we need to know**: [Specific question from NEEDS CLARIFICATION marker]
           
           **Suggested Answers**:
           
           | Option | Answer | Implications |
           |--------|--------|--------------|
           | A      | [First suggested answer] | [What this means for the feature] |
           | B      | [Second suggested answer] | [What this means for the feature] |
           | C      | [Third suggested answer] | [What this means for the feature] |
           | Custom | Provide your own answer | [Explain how to provide custom input] |
           
           **Your choice**: _[Wait for user response]_
           ```

        4. **CRITICAL - Table Formatting**: Ensure markdown tables are properly formatted:
           - Use consistent spacing with pipes aligned
           - Each cell should have spaces around content: `| Content |` not `|Content|`
           - Header separator must have at least 3 dashes: `|--------|`
           - Test that the table renders correctly in markdown preview
        5. Number questions sequentially (Q1, Q2, Q3 - max 3 total)
        6. Present all questions together before waiting for responses
        7. Wait for user to respond with their choices for all questions (e.g., "Q1: A, Q2: Custom - [details], Q3: B")
        8. Update the spec by replacing each [NEEDS CLARIFICATION] marker with the user's selected or provided answer
        9. Re-run validation after all clarifications are resolved

    d. **Update Checklist**: After each validation iteration, update the checklist file with current pass/fail status

     **Spec Mode Validation:**
     - **Comprehensive Checklist**: Full requirements quality validation
     - **Multiple Iterations**: Allow up to 3 clarification rounds for complex features
     - **Detailed Validation**: Check all requirement quality dimensions
     - **Success Criteria**: All requirements are clear, complete, and testable

8. **Context Population** (mode-aware):
     - **Read the generated spec.md** and extract key information
     - **Update context.md** with derived values instead of [NEEDS INPUT] placeholders:
       - **Feature**: Use the feature title/name from spec.md header
       - **Mission**: Extract the core purpose/goal from the feature description
       - **Code Paths**: Identify relevant codebase locations based on feature type and requirements
       - **Directives**: Reference applicable team directives from constitution/memory
       - **Research**: List any external research needs identified during specification
       - **Gateway**: Gateway or entry point for the feature
     - **Build Mode**: Populate Feature and Mission (minimum required)
     - **Spec Mode**: Populate all 6 fields with detailed, accurate values
     - **Validation**: Ensure no [NEEDS INPUT] markers remain in context.md

9. Report completion with branch name, spec file path, checklist results, and readiness for the next phase:
     - **Build Mode**: Ready for `/cx-spec.implement` (skip clarify/plan for lightweight execution)
     - **Spec Mode**: Ready for `/cx-spec.clarify` or `/cx-spec.plan`

10. **Mode Guidance**:
    - **Build Mode**: This mode prioritizes speed over completeness. Use `--mode=build` during specification for rapid prototyping.
    - **Spec Mode**: This mode provides thorough validation. Use `--mode=spec` (default) during specification for comprehensive planning.
    - **Changing Modes**: Create a new feature with the desired mode using `/cx-spec.specify --mode=build|spec` rather than trying to change an existing feature's mode.

**NOTE:** The script creates and checks out the new branch and initializes the spec file before writing.

## General Guidelines

## Quick Guidelines

- Focus on **WHAT** users need and **WHY**.
- Avoid HOW to implement (no tech stack, APIs, code structure).
- Written for business stakeholders, not developers.
- DO NOT create any checklists that are embedded in the spec. That will be a separate command.

### Section Requirements

- **Mandatory sections**: Must be completed for every feature
- **Optional sections**: Include only when relevant to the feature
- When a section doesn't apply, remove it entirely (don't leave as "N/A")

### For AI Generation

When creating this spec from a user prompt:

1. **Make informed guesses**: Use context, industry standards, and common patterns to fill gaps
2. **Document assumptions**: Record reasonable defaults in the Assumptions section
3. **Limit clarifications**: Maximum 3 [NEEDS CLARIFICATION] markers - use only for critical decisions that:
   - Significantly impact feature scope or user experience
   - Have multiple reasonable interpretations with different implications
   - Lack any reasonable default
4. **Prioritize clarifications**: scope > security/privacy > user experience > technical details
5. **Think like a tester**: Every vague requirement should fail the "testable and unambiguous" checklist item
6. **Common areas needing clarification** (only if no reasonable default exists):
   - Feature scope and boundaries (include/exclude specific use cases)
   - User types and permissions (if multiple conflicting interpretations possible)
   - Security/compliance requirements (when legally/financially significant)

**Examples of reasonable defaults** (don't ask about these):

- Data retention: Industry-standard practices for the domain
- Performance targets: Standard web/mobile app expectations unless specified
- Error handling: User-friendly messages with appropriate fallbacks
- Authentication method: Standard session-based or OAuth2 for web apps
- Integration patterns: RESTful APIs unless specified otherwise

### Success Criteria Guidelines

Success criteria must be:

1. **Measurable**: Include specific metrics (time, percentage, count, rate)
2. **Technology-agnostic**: No mention of frameworks, languages, databases, or tools
3. **User-focused**: Describe outcomes from user/business perspective, not system internals
4. **Verifiable**: Can be tested/validated without knowing implementation details

**Good examples**:

- "Users can complete checkout in under 3 minutes"
- "System supports 10,000 concurrent users"
- "95% of searches return results in under 1 second"
- "Task completion rate improves by 40%"

**Bad examples** (implementation-focused):

- "API response time is under 200ms" (too technical, use "Users see results instantly")
- "Database can handle 1000 TPS" (implementation detail, use user-facing metric)
- "React components render efficiently" (framework-specific)
- "Redis cache hit rate above 80%" (technology-specific)
