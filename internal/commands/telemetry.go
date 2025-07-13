package commands

import (
	"github.com/MakeNowJust/heredoc"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"time"
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

func telemetryAISubCommand(TelemetryAIWrapper wrappers.TelemetryWrapper) *cobra.Command {
	telemetryAICmd := &cobra.Command{
		Use:   "ai",
		Short: "telemetry for user events related to AI functionality.",
		Long:  "Collects telemetry data for user interactions related to AI features.",
		Example: heredoc.Doc(
			`
			$ cx telemetry ai --ai-provider <AI Provider> --timestamp <2025-07-10T08:30:00Z> --problem-severity <Problem Severity> --click-type<Click Type> --agent <agent> --engine <engine>
		`,
		),

		RunE: runTelemetryAI(TelemetryAIWrapper),
	}

	telemetryAICmd.PersistentFlags().String(params.AiProviderFlag, "", "AI Provider")
	telemetryAICmd.PersistentFlags().String(params.TimestampFlag, "", "Timestamp")
	telemetryAICmd.PersistentFlags().String(params.ProblemSeverityFlag, "", "Problem Severity")
	telemetryAICmd.PersistentFlags().String(params.ClickTypeFlag, "", "Click Type")
	telemetryAICmd.PersistentFlags().String(params.EngineFlag, "", "Engine")
	telemetryAICmd.PersistentFlags().String(params.AgentFlag, "", "Agent")

	return telemetryAICmd
}

func runTelemetryAI(TelemetryWrapper wrappers.TelemetryWrapper) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {

		aiProvider, _ := cmd.Flags().GetString("ai-provider")
		timestampStr, _ := cmd.Flags().GetString("timestamp")
		problemSeverity, _ := cmd.Flags().GetString("problem-severity")
		clickType, _ := cmd.Flags().GetString("click-type")
		agent, _ := cmd.Flags().GetString("agent")
		engine, _ := cmd.Flags().GetString("engine")

		var timestamp time.Time
		var err error
		if timestampStr == "" {
			timestamp = time.Now().UTC()
		} else {
			timestamp, err = time.Parse("2006-01-02T15:04:05Z07:00", timestampStr)
			if err != nil {
				return errors.Wrap(err, "Invalid timestamp format")
			}
		}

		err = TelemetryWrapper.SendDataToLog(wrappers.DataForAITelemetry{
			AIProvider:      aiProvider,
			ProblemSeverity: problemSeverity,
			ClickType:       clickType,
			Agent:           agent,
			Engine:          engine,
			Timestamp:       timestamp,
		})

		if err != nil {
			return errors.Wrapf(err, "%s", "Failed logging the data")
		}

		return nil
	}
}
