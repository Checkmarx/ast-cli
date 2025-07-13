package commands

import (
	"time"

	"github.com/MakeNowJust/heredoc"
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

func telemetryAISubCommand(TelemetryAIWrapper wrappers.TelemetryWrapper) *cobra.Command {
	telemetryAICmd := &cobra.Command{
		Use:   "ai",
		Short: "telemetry for user events related to AI functionality.",
		Long:  "Collects telemetry data for user interactions related to AI features.",
		Example: heredoc.Doc(
			`
			$ cx telemetry ai --ai-provider <AI Provider> --timestamp <2025-07-10T08:30:00Z> --problem-severity <Problem Severity> --type<Event Type> --sub-type<Event Name> --agent <Agent> --engine <Engine>
		`,
		),

		RunE: runTelemetryAI(TelemetryAIWrapper),
	}

	telemetryAICmd.PersistentFlags().String(params.AiProviderFlag, "", "AI Provider")
	telemetryAICmd.PersistentFlags().String(params.TimestampFlag, "", "Timestamp")
	telemetryAICmd.PersistentFlags().String(params.ProblemSeverityFlag, "", "Problem Severity")
	telemetryAICmd.PersistentFlags().String(params.TypeFlag, "", "Type")
	telemetryAICmd.PersistentFlags().String(params.SubTypeFlag, "", "Sub Type")
	telemetryAICmd.PersistentFlags().String(params.EngineFlag, "", "Engine")
	telemetryAICmd.PersistentFlags().String(params.AgentFlag, "", "Agent")

	return telemetryAICmd
}

func runTelemetryAI(telemetryWrapper wrappers.TelemetryWrapper) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		aiProvider, _ := cmd.Flags().GetString("ai-provider")
		timestampStr, _ := cmd.Flags().GetString("timestamp")
		problemSeverity, _ := cmd.Flags().GetString("problem-severity")
		eventType, _ := cmd.Flags().GetString("type")
		subType, _ := cmd.Flags().GetString("sub-type")
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

		err = telemetryWrapper.SendAIDataToLog(&wrappers.DataForAITelemetry{
			AIProvider:      aiProvider,
			ProblemSeverity: problemSeverity,
			Type:            eventType,
			SubType:         subType,
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
