package commands

import (
	"fmt"
	"strings"
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

	poolsCmd := &cobra.Command{
		Use:   "pools",
		Short: "Manage sast pools",
	}

	poolsListCmd := &cobra.Command{
		Use:   "list",
		Short: "List sast engine pools",
		RunE:  rm.RunListPoolsCommand,
	}

	poolCreateCmd := &cobra.Command{
		Use:   "create",
		Short: "Create sast engine pool",
		RunE:  rm.RunAddPoolCommand,
	}

	poolCreateCmd.PersistentFlags().StringP("description", "d", "",
		"Pool description")

	poolDeleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete sast engine pool",
		RunE:  rm.RunDeletePoolCommand,
	}

	poolDeleteCmd.PersistentFlags().StringP("id", "", "",
		"Pool id")

	poolProjectsCmd := &cobra.Command{
		Use: "projects",
	}

	poolProjectGetCmd := &cobra.Command{
		Use:   "get",
		Short: "List sast engine pool projects",
		RunE:  rm.RunGetPoolProjectsCommand,
	}
	poolProjectGetCmd.PersistentFlags().StringP("id", "", "",
		"Pool id")

	poolProjectsSetCmd := &cobra.Command{
		Use:   "set",
		Short: "Assigns projects to sast engine pool",
		RunE:  rm.RunSetPoolProjectsCommand,
	}

	poolProjectsSetCmd.PersistentFlags().StringP("id", "", "",
		"Pool id")

	poolProjectsCmd.AddCommand(poolProjectGetCmd, poolProjectsSetCmd)

	poolProjectTagsCmd := &cobra.Command{
		Use: "project-tags",
	}

	poolProjectTagsGetCmd := &cobra.Command{
		Use:   "get",
		Short: "List sast engine pool projects",
		RunE:  rm.RunGetPoolProjectTagsCommand,
	}
	poolProjectTagsGetCmd.PersistentFlags().StringP("id", "", "",
		"Pool id")

	poolProjectTagsSetCmd := &cobra.Command{
		Use:   "set",
		Short: "Assigns projects to sast engine pool",
		RunE:  rm.RunSetProjectTagsToPoolCommand,
	}
	poolProjectTagsSetCmd.PersistentFlags().StringP("id", "", "",
		"Pool id")

	poolProjectTagsCmd.AddCommand(poolProjectTagsGetCmd, poolProjectTagsSetCmd)

	poolEnginesCmd := &cobra.Command{
		Use: "Engines",
	}

	poolEngineGetCmd := &cobra.Command{
		Use:   "get",
		Short: "List sast engine pool Engines",
		RunE:  rm.RunGetPoolEnginesCommand,
	}
	poolEngineGetCmd.PersistentFlags().StringP("id", "", "",
		"Pool id")

	poolEnginesSetCmd := &cobra.Command{
		Use:   "set",
		Short: "Assigns Engines to sast engine pool",
		RunE:  rm.RunSetEnginesToPoolCommand,
	}

	poolEnginesSetCmd.PersistentFlags().StringP("id", "", "",
		"Pool id")

	poolEnginesCmd.AddCommand(poolEngineGetCmd, poolEnginesSetCmd)

	poolEngineTagsCmd := &cobra.Command{
		Use: "Engine-tags",
	}

	poolEngineTagsGetCmd := &cobra.Command{
		Use:   "get",
		Short: "List sast engine pool Engines",
		RunE:  rm.RunGetPoolEngineTagsCommand,
	}
	poolEngineTagsGetCmd.PersistentFlags().StringP("id", "", "",
		"Pool id")

	poolEngineTagsSetCmd := &cobra.Command{
		Use:   "set",
		Short: "Assigns Engines to sast engine pool",
		RunE:  rm.RunSetEngineTagsToPoolCommand,
	}
	poolEngineTagsSetCmd.PersistentFlags().StringP("id", "", "",
		"Pool id")

	poolEngineTagsCmd.AddCommand(poolEngineTagsGetCmd, poolEngineTagsSetCmd)

	poolsCmd.AddCommand(poolsListCmd, poolCreateCmd, poolDeleteCmd, poolProjectsCmd, poolProjectTagsCmd, poolEnginesCmd, poolEngineTagsCmd)

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

	addFormatFlagToMultipleCommands([]*cobra.Command{scansCmd, enginesCmd, statsCmd, poolsCmd},
		formatTable, formatJSON, formatList)
	sastrmCmd.AddCommand(scansCmd, enginesCmd, statsCmd, poolsCmd)
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
	return printByFormat(cmd, scanViews(scans))
}

func (c rmCommands) RunEnginesCommand(cmd *cobra.Command, args []string) error {
	PrintIfVerbose("Reading sast resources engines")
	engines, err := c.rmWrapper.GetEngines()
	if err != nil {
		return errors.Wrap(err, "failed get engines")
	}
	return printByFormat(cmd, engineViews(engines))
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

	return printByFormat(cmd, stats)
}

func (c rmCommands) RunAddPoolCommand(cmd *cobra.Command, args []string) error {
	description := cmd.Flag("description").Value.String()
	PrintIfVerbose(fmt.Sprintf("Creating pool description:%s", description))
	pool, err := c.rmWrapper.AddPool(description)
	PrintIfVerbose("Pool created")
	if err != nil {
		return err
	}
	return printByFormat(cmd, pool)
}

func (c rmCommands) RunDeletePoolCommand(cmd *cobra.Command, args []string) error {
	id := cmd.Flag("id").Value.String()
	PrintIfVerbose(fmt.Sprintf("Deleting pool id:%s", id))
	err := c.rmWrapper.DeletePool(id)
	if err != nil {
		return err
	}
	PrintIfVerbose("Pool deleted")
	return nil
}

func (c rmCommands) RunListPoolsCommand(cmd *cobra.Command, args []string) error {
	PrintIfVerbose("Getting pools")
	pools, err := c.rmWrapper.GetPools()
	if err != nil {
		return err
	}
	return printByFormat(cmd, pools)
}

func (c rmCommands) RunGetPoolEnginesCommand(cmd *cobra.Command, args []string) error {
	id := cmd.Flag("id").Value.String()
	PrintIfVerbose(fmt.Sprintf("Getting pool engines poolID:%s", id))
	pools, err := c.rmWrapper.GetPools()
	if err != nil {
		return err
	}
	return printByFormat(cmd, pools)
}

func (c rmCommands) RunGetPoolEngineTagsCommand(cmd *cobra.Command, args []string) error {
	id := cmd.Flag("id").Value.String()
	PrintIfVerbose(fmt.Sprintf("Getting pool engine tags poolID:%s", id))
	tags, err := c.rmWrapper.GetPoolEngineTags(id)
	if err != nil {
		return err
	}
	return printByFormat(cmd, tags)
}

func (c rmCommands) RunGetPoolProjectsCommand(cmd *cobra.Command, args []string) error {
	id := cmd.Flag("id").Value.String()
	PrintIfVerbose(fmt.Sprintf("Getting pool projects poolID:%s", id))
	projects, err := c.rmWrapper.GetPoolProjects(id)
	if err != nil {
		return err
	}
	return printByFormat(cmd, projects)
}

func (c rmCommands) RunGetPoolProjectTagsCommand(cmd *cobra.Command, args []string) error {
	id := cmd.Flag("id").Value.String()
	PrintIfVerbose(fmt.Sprintf("Getting pool project tags poolID:%s", id))
	tags, err := c.rmWrapper.GetPoolProjectTags(id)
	if err != nil {
		return err
	}
	return printByFormat(cmd, tags)
}

func (c rmCommands) RunSetEnginesToPoolCommand(cmd *cobra.Command, args []string) error {
	id := cmd.Flag("id").Value.String()
	return c.rmWrapper.SetPoolEngines(id, args)
}
func (c rmCommands) RunSetPoolProjectsCommand(cmd *cobra.Command, args []string) error {
	id := cmd.Flag("id").Value.String()
	return c.rmWrapper.SetPoolProjects(id, args)
}

func (c rmCommands) RunSetEngineTagsToPoolCommand(cmd *cobra.Command, args []string) error {
	id := cmd.Flag("id").Value.String()
	tags, err := parseTags(args)
	if err != nil {
		return err
	}
	return c.rmWrapper.SetPoolEngineTags(id, tags)
}

func (c rmCommands) RunSetProjectTagsToPoolCommand(cmd *cobra.Command, args []string) error {
	id := cmd.Flag("id").Value.String()
	tags, err := parseTags(args)
	if err != nil {
		return err
	}
	return c.rmWrapper.SetPoolProjectTags(id, tags)
}

func parseTags(args []string) (tags []wrappers.Tag, err error) {
	for _, arg := range args {
		parts := strings.Split(arg, "=")
		if len(parts) != 2 {
			return nil, errors.New("provide tags in key=value format")
		}
		tags = append(tags, wrappers.Tag{
			Key:   parts[0],
			Value: parts[1],
		})
	}
	return
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
	QueuedAt   time.Time         `format:"time:01-02-06 15:04:05.000;name:Queued at" json:"queued-at"`
	RunningAt  *time.Time        `format:"time:01-02-06 15:04:05.000;name:Running at" json:"running-at"`
	Engine     string            `format:"engine"`
	Properties map[string]string `json:"properties"`
}

type rmEngineView struct {
	ID           string            `json:"id"`
	Status       string            `json:"status"`
	ScanID       string            `json:"scan"`
	RegisteredAt time.Time         `format:"time:01-02-06 15:04:05.000;name:Discovered at" json:"registered-at"`
	UpdatedAt    time.Time         `format:"time:01-02-06 15:04:05.000;name:Heartbeat at" json:"updated-at"`
	Properties   map[string]string `json:"properties"`
	Tags         map[string]string `json:"tags"`
}
