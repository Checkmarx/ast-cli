package commands

import (
	"github.com/MakeNowJust/heredoc"
	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
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
			$ cx telemetry ai --ai-provider <AI Provider> --problem-severity <Problem Severity> --type<Event Type> --sub-type<Event Name> --agent <Agent> --engine <Engine>
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
	telemetryAICmd.PersistentFlags().Int(params.TotalCountFlag, 0, "Total Count")

	return telemetryAICmd
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
		totalCount, _ := cmd.Flags().GetInt("total-count")
		uniqueId := wrappers.GetUniqueID()
		logger.PrintIfVerbose("unique id: " + uniqueId)
		err := telemetryWrapper.SendAIDataToLog(&wrappers.DataForAITelemetry{
			AIProvider:      aiProvider,
			ProblemSeverity: problemSeverity,
			Type:            eventType,
			SubType:         subType,
			Agent:           agent,
			Engine:          engine,
			ScanType:        scanType,
			Status:          status,
			TotalCount:      totalCount,
			UniqueID:        uniqueId,
		})

		if err != nil {
			return errors.Wrapf(err, "%s", "Failed logging the data")
		}

		return nil
	}
}
