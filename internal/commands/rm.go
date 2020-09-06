package commands

import (
	"fmt"
	"time"

	"github.com/checkmarxDev/sast-rm/pkg/api/rest"

	"github.com/checkmarxDev/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewSastResourcesCommand(rmWrapper wrappers.SastRmWrapper) *cobra.Command {
	rm := rmCommands{rmWrapper}

	scansCmd := &cobra.Command{
		Use:   "scans",
		Short: "Display scans in sast queue",
		RunE:  rm.RunScansCommand,
	}

	enginesCmd := &cobra.Command{
		Use:   "engines",
		Short: "Display sast engines",
		RunE:  rm.RunEnginesCommand,
	}

	statsCmd := &cobra.Command{
		Use:   "stats",
		Short: "Display sast queue statistics",
		RunE:  rm.RunStatsCommand,
	}

	statsCmd.PersistentFlags().StringP("resolution", "r", "moment",
		"Resolution, one of: minute, hour, day, week, moment")

	sastrmCmd := &cobra.Command{
		Use:   "sast-rm",
		Short: "SAST resource management",
	}
	sastrmCmd.AddCommand(scansCmd, enginesCmd, statsCmd)
	return sastrmCmd
}

type rmCommands struct {
	rmWrapper wrappers.SastRmWrapper
}

func (c rmCommands) RunScansCommand(cmd *cobra.Command, args []string) error {
	PrintIfVerbose("Reading sast resources scans")
	scans, err := c.rmWrapper.GetScans()
	if err != nil {
		return err
	}
	return Print(cmd.OutOrStdout(), scanViews(scans))
}

func (c rmCommands) RunEnginesCommand(cmd *cobra.Command, args []string) error {
	PrintIfVerbose("Reading sast resources engines")
	engines, err := c.rmWrapper.GetEngines()
	if err != nil {
		return errors.Wrap(err, "failed get engines")
	}
	return Print(cmd.OutOrStdout(), engineViews(engines))
}

func (c rmCommands) RunStatsCommand(cmd *cobra.Command, args []string) error {
	resolutionName := cmd.Flag("resolution").Value.String()
	resolution, ok := wrappers.StatResolutions[resolutionName]
	if !ok {
		return errors.Errorf("unknown resolution %s", resolutionName)
	}
	PrintIfVerbose(fmt.Sprintf("Reading sast resources statistics per %s", resolution))
	stats, err := c.rmWrapper.GetStats(resolution)
	if err != nil {
		return err
	}

	return Print(cmd.OutOrStdout(), stats)
}

func scanViews(scans []*rest.Scan) []*rmScanView {
	result := make([]*rmScanView, 0, len(scans))
	for _, s := range scans {
		result = append(result, &rmScanView{
			ID:         s.ID,
			State:      string(s.State),
			QueuedAt:   s.QueuedAt,
			RunningAt:  s.RunningAt,
			Engine:     s.Engine,
			Properties: s.Properties,
		})
	}
	return result
}

func engineViews(engines []*rest.Engine) []*rmEngineView {
	result := make([]*rmEngineView, 0, len(engines))
	for _, w := range engines {
		result = append(result, &rmEngineView{
			ID:           w.ID,
			Status:       string(w.Status),
			ScanID:       w.ScanID,
			RegisteredAt: w.RegisteredAt,
			UpdatedAt:    w.UpdatedAt,
			Properties:   w.Properties,
		})
	}
	return result
}

type rmScanView struct {
	ID         string            `json:"id"`
	State      string            `json:"state"`
	QueuedAt   time.Time         `format:"time:06-01-02 15:04:05.000;name:Queued at" json:"queued-at"`
	RunningAt  *time.Time        `format:"time:06-01-02 15:04:05.000;name:Running at" json:"running-at"`
	Engine     string            `json:"engine"`
	Properties map[string]string `json:"properties"`
}

type rmEngineView struct {
	ID           string            `json:"id"`
	Status       string            `json:"status"`
	ScanID       string            `json:"scan"`
	RegisteredAt time.Time         `format:"time:06-01-02 15:04:05.000;name:Discovered at" json:"registered-at"`
	UpdatedAt    time.Time         `format:"time:06-01-02 15:04:05.000;name:Heartbeat at" json:"updated-at"`
	Properties   map[string]string `json:"properties"`
}
