package secrets

import (
	"fmt"

	agenthooks "github.com/Checkmarx/ast-cx-hooks"
	"github.com/checkmarx/ast-cli/internal/commands/agenthooks/agentprofile"
	"github.com/checkmarx/ast-cli/internal/commands/agenthooks/cursorplugin"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/secretsrealtime"
)

func additionalContext(filePath, cxBinary string, findings []secretsrealtime.SecretsRealtimeResult, workDir string, agent agenthooks.AgentID, sessionID string) string {
	codeTool := "mcp__Checkmarx__codeRemediation"
	if agent == agenthooks.AgentGemini {
		codeTool = "mcp_Checkmarx_codeRemediation"
	}
	skill := secretsSkillName(agent)
	suppressCmds := secretsSuppressCommands(cxBinary, findings, workDir, agent, sessionID)
	return fmt.Sprintf(
		"Checkmarx secret detection found hardcoded secrets in %s. "+
			"Do not bypass the scan by writing the same content through another tool or shell command.\n"+
			"Action required:\n"+
			"  1. For each finding, invoke the %s skill — "+
			"the findings are already in context so it will skip the scan and go directly to "+
			"MCP-driven remediation; the skill also handles MCP unavailability and self-recovery.\n"+
			"  2. If that skill is not available in this session, %s\n"+
			"     This is the only supported remediation path — do not write your own fix if the MCP is unavailable.\n"+
			"  3. If %s is unavailable, tell the user to reconnect the\n"+
			"     Checkmarx MCP (%s), then retry. Do not proceed until the MCP is available.\n"+
			"  4. ANALYZE each finding. A placeholder, example, or test fixture can be a false positive. "+
			"If the user confirms a false positive, suppress it by running the corresponding command\n"+
			"     below, then retry the write:\n%s",
		filePath, skill, remediationInstructions(codeTool),
		codeTool, agentprofile.McpReconnect(agentLabel(agent)), suppressCmds,
	)
}

func remediationInstructions(codeTool string) string {
	return fmt.Sprintf("For each finding, call the %s tool with:\n"+
		"  {\n"+
		"    \"type\": \"secrets\",\n"+
		"    \"metadata\": {\n"+
		"      \"ruleId\": \"[Title from finding]\",\n"+
		"      \"description\": \"[Description from finding]\",\n"+
		"      \"line\": \"[line from finding]\"\n"+
		"    }\n"+
		"  }\n"+
		"Apply the remediation guidance the tool returns: replace the hardcoded secret with an "+
		"environment-variable reference, including any Safe Refactor of other usages the tool returns. "+
		"Then retry the write so the hook can verify the secret is gone. Do not echo the secret value.", codeTool)
}

func cursorAdditionalContext(filePath, cxBinary string, findings []secretsrealtime.SecretsRealtimeResult, workDir, sessionID string) string {
	suppressCmds := secretsSuppressCommands(cxBinary, findings, workDir, agenthooks.AgentCursor, sessionID)
	skill := secretsSkillName(agenthooks.AgentCursor)
	return fmt.Sprintf(
		"Checkmarx secret detection found hardcoded secrets in %s. "+
			"Do not bypass the scan by writing the same content through another tool or shell command. "+
			"ANALYZE each finding to determine if it is a real secret or a false positive "+
			"(for example a placeholder, example, or test fixture). "+
			"Follow the cx-hook-deny.mdc rule for this deny. "+
			"ASK THE USER FIRST, for every real finding, before taking any action: \"A hardcoded secret "+
			"was detected. Would you like to remediate it (replace it with an environment-variable reference via MCP) "+
			"or suppress it (mark as a confirmed false positive and unblock the write)?\" and wait for "+
			"their answer. Do not decide this yourself — an intentionally-inserted secret (e.g. "+
			"in a lab/demo/training file the user asked for on purpose) is NOT the same as a confirmed "+
			"false positive: suppress only on the user's explicit instruction, never because the "+
			"request seems intentional. "+
			"Apply the cx-devassist-secrets.mdc rule: for each finding the user asks you to remediate, "+
			"invoke the %s skill exactly as written — do not skip, abbreviate, or reimplement its steps "+
			"inline. The findings are already in context so it will skip the scan and go directly to "+
			"MCP-driven remediation; the skill also handles MCP unavailability and self-recovery. "+
			"Always show its Secret Remediation Summary to the user verbatim when done. "+
			"If that skill is not available in this session, %s\n"+
			"If the user chooses to suppress a finding, run the corresponding command below, then retry the write:\n%s",
		filePath, skill, remediationInstructions(cursorplugin.MCPTool("codeRemediation")),
		suppressCmds,
	)
}

func secretsSkillName(agent agenthooks.AgentID) string {
	if agent == agenthooks.AgentGemini {
		return "/cx-devassist-secrets"
	}
	return "cx-devassist:cx-devassist-secrets"
}
