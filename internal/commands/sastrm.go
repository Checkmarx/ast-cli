package commands

import (
	"encoding/json"
	"fmt"
	"time"

	rm "github.com/checkmarxDev/sast-rm/pkg/api/v1/rest"

	"github.com/checkmarxDev/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewSastRmCommand(queueWrapper wrappers.SastRmWrapper) *cobra.Command {

	queueScansCmd := &cobra.Command{
		Use:   "scans",
		Short: "Display SAST queue scans",
		RunE: func(cmd *cobra.Command, args []string) error {
			scans, err := queueWrapper.GetScans()
			if err != nil {
				return errors.Wrap(err, "failed get scans")
			}
			scansJson, _ := json.Marshal(scanDisplay(scans))
			fmt.Fprintln(cmd.OutOrStdout(), string(scansJson))
			return nil
		},
	}

	queueWorkersCmd := &cobra.Command{
		Use:   "workers",
		Short: "Display SAST queue workers",
		RunE: func(cmd *cobra.Command, args []string) error {
			workers, err := queueWrapper.GetWorkers()
			if err != nil {
				return errors.Wrap(err, "failed get workers")
			}
			workersJson, _ := json.Marshal(workerDisplay(workers))
			fmt.Fprintln(cmd.OutOrStdout(), string(workersJson))
			return nil
		},
	}

	queueStatusCmd := &cobra.Command{
		Use:   "status",
		Short: "Display SAST queue status",
		RunE: func(cmd *cobra.Command, args []string) error {
			workers, err := queueWrapper.GetWorkers()
			if err != nil {
				return errors.Wrap(err, "failed get workers")
			}
			scans, err := queueWrapper.GetScans()
			if err != nil {
				return errors.Wrap(err, "failed get scans")
			}
			statusJson, _ := json.Marshal(map[string]interface{}{
				"scans":   scanDisplay(scans),
				"workers": workerDisplay(workers),
			})
			fmt.Fprintln(cmd.OutOrStdout(), string(statusJson))
			return nil
		},
	}

	sastrmCmd := &cobra.Command{
		Use:   "sast-rm",
		Short: "Manage AST sast queue",
	}
	sastrmCmd.AddCommand(queueScansCmd, queueWorkersCmd, queueStatusCmd)
	return sastrmCmd
}

func scanDisplay(scans []*rm.Scan) []*ScanView {
	result := make([]*ScanView, 0, len(scans))
	for _, s := range scans {
		result = append(result, &ScanView{
			Id:         short8(&s.Id),
			State:      string(s.State),
			Priority:   s.Priority,
			QueuedAt:   formatTime(&s.QueuedAt),
			RunningAt:  formatTime(s.RunningAt),
			Worker:     short(&s.Worker, 11),
			Constrains: s.Constrains,
		})
	}
	return result
}

func workerDisplay(workers []*rm.Worker) []*WorkerView {
	result := make([]*WorkerView, 0, len(workers))
	for _, w := range workers {
		result = append(result, &WorkerView{
			Id:           short(&w.ID, 11),
			Status:       string(w.Status),
			TaskId:       short8(&w.TaskID),
			RegisteredAt: formatTime(&w.RegisteredAt),
			UpdatedAt:    formatTime(&w.UpdatedAt),
			Properties:   w.Properties,
		})
	}
	return result
}

func formatTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("06-01-02 15:04:05.111")
}

func short8(id *string) string {
	if id == nil || *id == "" {
		return ""
	}
	if len(*id) < 8 {
		return *id
	}
	return (*id)[0:8]
}

func short(id *string, length int) string {
	if id == nil || *id == "" {
		return ""
	}
	if len(*id) < length {
		return *id
	}
	return (*id)[0:length]
}

type ScanView struct {
	Id         string            `json:"id"`
	State      string            `json:"state"`
	Priority   float32           `json:"priority,"`
	QueuedAt   string            `json:"queued-at"`
	RunningAt  string            `json:"running-at"`
	Worker     string            `json:"worker"`
	Constrains map[string]string `json:"constrains"`
}

type WorkerView struct {
	Id           string            `json:"id"`
	Status       string            `json:"status"`
	TaskId       string            `json:"scan"`
	RegisteredAt string            `json:"registered-at"`
	UpdatedAt    string            `json:"updated-at"`
	Properties   map[string]string `json:"properties"`
}
