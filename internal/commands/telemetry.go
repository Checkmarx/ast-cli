package commands

import (
	"strings"

	"github.com/MakeNowJust/heredoc"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewTelemetryCommand(telemetryWrapper wrappers.TelemetryWrapper) *cobra.Command {
	telemetryCmd := &cobra.Command{
		Use:   "telemetry",
		Short: "Telemetry user events",
		Long:  "The 'telemetry' command allows collecting and sending user interaction events for analysis purposes.",
	}
	telemetryAICmd := telemetryAISubCommand(telemetryWrapper)

	telemetryCmd.AddCommand(telemetryAICmd)
	return telemetryCmd
}

func telemetryAISubCommand(telemetryAIWrapper wrappers.TelemetryWrapper) *cobra.Command {
	telemetryAICmd := &cobra.Command{
		Use:   "ai",
		Short: "telemetry for user events related to AI functionality.",
		Long:  "Collects telemetry data for user interactions related to AI features.",
		Example: heredoc.Doc(
			`
			$ cx telemetry ai --ai-provider <AI Provider> --problem-severity <Problem Severity> --type<Event Type> --sub-type<Event Name> --agent <Agent> --engine <Engine> --event-attributes "apiVersion=v2;vulnerabilityId=CVE-2021-44228;remediationMode=DevAssist"
		`,
		),

		RunE: runTelemetryAI(telemetryAIWrapper),
	}

	telemetryAICmd.PersistentFlags().String(params.AiProviderFlag, "", "AI Provider")
	telemetryAICmd.PersistentFlags().String(params.ProblemSeverityFlag, "", "Problem Severity")
	telemetryAICmd.PersistentFlags().String(params.TypeFlag, "", "Type")
	telemetryAICmd.PersistentFlags().String(params.SubTypeFlag, "", "Sub Type")
	telemetryAICmd.PersistentFlags().String(params.EngineFlag, "", "Engine")
	telemetryAICmd.PersistentFlags().String(params.AgentFlag, "", "Agent")
	telemetryAICmd.PersistentFlags().String(params.ScanTypeFlag, "", "Scan Type")
	telemetryAICmd.PersistentFlags().String(params.StatusFlag, "", "Status")
	telemetryAICmd.PersistentFlags().String(params.StatusResultFlag, "", "Status Result")
	telemetryAICmd.PersistentFlags().Int(params.TotalCountFlag, 0, "Total Count")
	telemetryAICmd.PersistentFlags().String(params.AgentVersionFlag, "", "Agent Version")
	telemetryAICmd.PersistentFlags().String(params.CxPluginVersionFlag, "", "CX Plugin Version")
	telemetryAICmd.PersistentFlags().String(params.PluginTypeFlag, "", "Plugin Type")
	telemetryAICmd.PersistentFlags().String(params.RepoURLFlag, "", "Repository URL")
	telemetryAICmd.PersistentFlags().String(params.EventAttributesFlag, "", "Event Attributes in key=value;key=value format (e.g., apiVersion=v2;vulnerabilityId=CVE-2021-44228;remediationMode=DevAssist)")
	telemetryAICmd.PersistentFlags().String(params.ApiVersionFlag, "", "API Version")

	return telemetryAICmd
}

func parseEventAttributes(eventAttributesStr string) *wrappers.EventAttributes {
	if eventAttributesStr == "" {
		return nil
	}

	attrs := make(wrappers.EventAttributes)
	pairs := strings.Split(eventAttributesStr, ";")

	for _, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) != 2 {
			continue
		}

		key := strings.TrimSpace(kv[0])
		value := strings.TrimSpace(kv[1])

		if key != "" {
			attrs[key] = value
		}
	}

	if len(attrs) == 0 {
		return nil
	}

	return &attrs
}

func runTelemetryAI(telemetryWrapper wrappers.TelemetryWrapper) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		aiProvider, _ := cmd.Flags().GetString("ai-provider")
		problemSeverity, _ := cmd.Flags().GetString("problem-severity")
		eventType, _ := cmd.Flags().GetString("type")
		subType, _ := cmd.Flags().GetString("sub-type")
		agent, _ := cmd.Flags().GetString("agent")
		engine, _ := cmd.Flags().GetString("engine")
		scanType, _ := cmd.Flags().GetString("scan-type")
		status, _ := cmd.Flags().GetString("status")
		totalCount, _ := cmd.Flags().GetInt(params.TotalCountFlag)
		statusResult, _ := cmd.Flags().GetString(params.StatusResultFlag)
		agentVersion, _ := cmd.Flags().GetString(params.AgentVersionFlag)
		cxPluginVersion, _ := cmd.Flags().GetString(params.CxPluginVersionFlag)
		pluginType, _ := cmd.Flags().GetString(params.PluginTypeFlag)
		repoURL, _ := cmd.Flags().GetString(params.RepoURLFlag)
		eventAttributesStr, _ := cmd.Flags().GetString(params.EventAttributesFlag)
		apiVersion, _ := cmd.Flags().GetString(params.ApiVersionFlag)

		uniqueID := wrappers.GetUniqueID()
		machineUniqueId := wrappers.GetMachineID()

		var sessionID string
		if eventType == "authentication" && status == "success" {
			sessionID = wrappers.GetSessionID()
		} else {
			sessionID = viper.GetString(params.SessionIDConfigKey)
		}

		eventAttrs := parseEventAttributes(eventAttributesStr)

		err := telemetryWrapper.SendAIDataToLog(&wrappers.DataForAITelemetry{
			AIProvider:      aiProvider,
			ProblemSeverity: problemSeverity,
			Type:            eventType,
			SubType:         subType,
			Agent:           agent,
			AgentVersion:    agentVersion,
			CxPluginVersion: cxPluginVersion,
			PluginType:      pluginType,
			Engine:          engine,
			ScanType:        scanType,
			Status:          status,
			StatusResult:    statusResult,
			APIVersion:      apiVersion,
			TotalCount:      totalCount,
			RepoURL:         repoURL,
			UniqueID:        uniqueID,
			SessionID:       sessionID,
			MachineUniqueId: machineUniqueId,
			EventAttributes: eventAttrs,
		})

		if err != nil {
			return errors.Wrapf(err, "%s", "Failed logging the data")
		}

		return nil
	}
}
