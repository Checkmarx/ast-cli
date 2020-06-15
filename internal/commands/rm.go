package commands

import (
	"fmt"
	"time"

	"github.com/checkmarxDev/sast-rm/pkg/api/v1/rest"

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
		Use:     "sast-resources",
		Aliases: []string{"sr"},
		Short:   "SAST queue status (short form: 'sr')",
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
	Print(cmd.OutOrStdout(), scanViews(scans))
	return nil
}

func (c rmCommands) RunEnginesCommand(cmd *cobra.Command, args []string) error {
	PrintIfVerbose("Reading sast resources engines")
	engines, err := c.rmWrapper.GetEngines()
	if err != nil {
		return errors.Wrap(err, "failed get engines")
	}
	Print(cmd.OutOrStdout(), engineViews(engines))
	return nil
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

	Print(cmd.OutOrStdout(), stats)
	return nil
}

func scanViews(scans []*rest.Scan) []*rmScanView {
	result := make([]*rmScanView, 0, len(scans))
	for _, s := range scans {
		result = append(result, &rmScanView{
			ID:         s.ID,
			State:      string(s.State),
			Priority:   s.Priority,
			QueuedAt:   s.QueuedAt,
			RunningAt:  s.RunningAt,
			Engine:     s.Engine,
			Constrains: s.Constrains,
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
	ID         string            `format:"maxlen:8" json:"id"`
	State      string            `json:"state"`
	Priority   float32           `json:"priority"`
	QueuedAt   time.Time         `format:"time:06-01-02 15:04:05.000;name:Queued at" json:"queued-at"`
	RunningAt  *time.Time        `format:"time:06-01-02 15:04:05.000;name:Running at" json:"running-at"`
	Engine     string            `format:"maxlen:13" json:"worker"`
	Constrains map[string]string `json:"constrains"`
}

type rmEngineView struct {
	ID           string            `format:"maxlen:13" json:"id"`
	Status       string            `json:"status"`
	ScanID       string            `format:"maxlen:8" json:"scan"`
	RegisteredAt time.Time         `format:"time:06-01-02 15:04:05.000;name:Discovered at" json:"registered-at"`
	UpdatedAt    time.Time         `format:"time:06-01-02 15:04:05.000;name:Heartbeat at" json:"updated-at"`
	Properties   map[string]string `json:"properties"`
}
