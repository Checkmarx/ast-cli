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
		Short: "Delete sast engine pools",
		RunE:  rm.RunDeletePoolCommand,
	}

	poolProjectsCmd := &cobra.Command{
		Use: "projects",
	}

	poolProjectGetCmd := &cobra.Command{
		Use:   "get",
		Short: "List sast engine pool projects",
		RunE:  rm.RunGetPoolProjectsCommand,
	}
	poolProjectGetCmd.PersistentFlags().StringP("pool-id", "i", "",
		"Pool id")

	poolProjectsSetCmd := &cobra.Command{
		Use:   "set",
		Short: "Assigns projects to sast engine pool",
		RunE:  rm.RunSetPoolProjectsCommand,
	}

	poolProjectsSetCmd.PersistentFlags().StringP("pool-id", "i", "",
		"Pool id")

	poolProjectsCmd.AddCommand(poolProjectGetCmd, poolProjectsSetCmd)

	poolProjectTagsCmd := &cobra.Command{
		Use: "project-tags",
	}

	poolProjectTagsGetCmd := &cobra.Command{
		Use:   "get",
		Short: "List sast engine pool project tags",
		RunE:  rm.RunGetPoolProjectTagsCommand,
	}
	poolProjectTagsGetCmd.PersistentFlags().StringP("pool-id", "i", "",
		"Pool id")

	poolProjectTagsSetCmd := &cobra.Command{
		Use:   "set",
		Short: "Assigns projects tags to sast engine pool",
		RunE:  rm.RunSetProjectTagsToPoolCommand,
	}
	poolProjectTagsSetCmd.PersistentFlags().StringP("pool-id", "i", "",
		"Pool id")

	poolProjectTagsCmd.AddCommand(poolProjectTagsGetCmd, poolProjectTagsSetCmd)

	poolEnginesCmd := &cobra.Command{
		Use: "engines",
	}

	poolEngineGetCmd := &cobra.Command{
		Use:   "get",
		Short: "List sast engine pool engines",
		RunE:  rm.RunGetPoolEnginesCommand,
	}
	poolEngineGetCmd.PersistentFlags().StringP("pool-id", "i", "",
		"Pool id")

	poolEnginesSetCmd := &cobra.Command{
		Use:   "set",
		Short: "Assigns engines to sast engine pool",
		RunE:  rm.RunSetEnginesToPoolCommand,
	}

	poolEnginesSetCmd.PersistentFlags().StringP("pool-id", "i", "",
		"Pool id")

	poolEnginesCmd.AddCommand(poolEngineGetCmd, poolEnginesSetCmd)

	poolEngineTagsCmd := &cobra.Command{
		Use: "engine-tags",
	}

	poolEngineTagsGetCmd := &cobra.Command{
		Use:   "get",
		Short: "List sast engine pool engine tags",
		RunE:  rm.RunGetPoolEngineTagsCommand,
	}
	poolEngineTagsGetCmd.PersistentFlags().StringP("pool-id", "i", "",
		"Pool id")

	poolEngineTagsSetCmd := &cobra.Command{
		Use:   "set",
		Short: "Assigns engines tags to sast engine pool",
		RunE:  rm.RunSetEngineTagsToPoolCommand,
	}
	poolEngineTagsSetCmd.PersistentFlags().StringP("pool-id", "i", "",
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
	if len(args) == 0 {
		return errors.New("no pool ids provided")
	}
	PrintIfVerbose(fmt.Sprintf("Deleting pools ids:%s", strings.Join(args, ", ")))
	for _, id := range args {
		err := c.rmWrapper.DeletePool(id)
		if err != nil {
			return err
		}
		PrintIfVerbose("Deleted pool id=" + id)
	}
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
	id := cmd.Flag("pool-id").Value.String()
	PrintIfVerbose(fmt.Sprintf("Getting pool engines poolID:%s", id))
	engines, err := c.rmWrapper.GetPoolEngines(id)
	if err != nil {
		return err
	}
	return printByFormat(cmd, elementViews(engines))
}

func (c rmCommands) RunGetPoolEngineTagsCommand(cmd *cobra.Command, args []string) error {
	id := cmd.Flag("pool-id").Value.String()
	PrintIfVerbose(fmt.Sprintf("Getting pool engine tags poolID:%s", id))
	tags, err := c.rmWrapper.GetPoolEngineTags(id)
	if err != nil {
		return err
	}
	return printByFormat(cmd, tagViews(tags))
}

func (c rmCommands) RunGetPoolProjectsCommand(cmd *cobra.Command, args []string) error {
	id := cmd.Flag("pool-id").Value.String()
	PrintIfVerbose(fmt.Sprintf("Getting pool projects poolID:%s", id))
	projects, err := c.rmWrapper.GetPoolProjects(id)
	if err != nil {
		return err
	}
	return printByFormat(cmd, elementViews(projects))
}

func (c rmCommands) RunGetPoolProjectTagsCommand(cmd *cobra.Command, args []string) error {
	id := cmd.Flag("pool-id").Value.String()
	PrintIfVerbose(fmt.Sprintf("Getting pool project tags poolID:%s", id))
	tags, err := c.rmWrapper.GetPoolProjectTags(id)
	if err != nil {
		return err
	}
	return printByFormat(cmd, tagViews(tags))
}

func (c rmCommands) RunSetEnginesToPoolCommand(cmd *cobra.Command, args []string) error {
	id := cmd.Flag("pool-id").Value.String()
	PrintIfVerbose(fmt.Sprintf("Setting pool engines, poolID:%s engines:%s",
		id, strings.Join(args, ", ")))
	return c.rmWrapper.SetPoolEngines(id, args)
}
func (c rmCommands) RunSetPoolProjectsCommand(cmd *cobra.Command, args []string) error {
	id := cmd.Flag("pool-id").Value.String()
	PrintIfVerbose(fmt.Sprintf("Setting pool projects, poolID:%s projects:%s",
		id, strings.Join(args, ", ")))
	return c.rmWrapper.SetPoolProjects(id, args)
}

func (c rmCommands) RunSetEngineTagsToPoolCommand(cmd *cobra.Command, args []string) error {
	id := cmd.Flag("pool-id").Value.String()
	tags, err := parseTags(args)
	if err != nil {
		return err
	}
	PrintIfVerbose(fmt.Sprintf("Setting pool engine tags, poolID:%s tags:%s",
		id, strings.Join(args, ", ")))
	return c.rmWrapper.SetPoolEngineTags(id, tags)
}

func (c rmCommands) RunSetProjectTagsToPoolCommand(cmd *cobra.Command, args []string) error {
	id := cmd.Flag("pool-id").Value.String()
	tags, err := parseTags(args)
	if err != nil {
		return err
	}
	PrintIfVerbose(fmt.Sprintf("Setting pool project tags, poolID:%s tags:%s",
		id, strings.Join(args, ", ")))
	return c.rmWrapper.SetPoolProjectTags(id, tags)
}

func parseTags(args []string) (map[string]string, error) {
	tags := map[string]string{}
	for _, arg := range args {
		parts := strings.Split(arg, "=")
		if len(parts) != 2 { //nolint:gomnd
			return nil, errors.New("provide tags in key=value format")
		}
		tags[parts[0]] = parts[1]
	}
	return tags, nil
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
			Tags:         w.Tags,
		})
	}
	return result
}

type rmScanView struct {
	ID         string            `json:"id"`
	State      string            `json:"state"`
	QueuedAt   time.Time         `format:"time:01-02-06 15:04:05.000;name:Queued at" json:"queued-at"`
	RunningAt  *time.Time        `format:"time:01-02-06 15:04:05.000;name:Running at" json:"running-at"`
	Engine     string            `json:"engine"`
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

func tagViews(tags map[string]string) []*tagView {
	result := make([]*tagView, 0, len(tags))
	for k, v := range tags {
		result = append(result, &tagView{Tag: k, Value: v})
	}
	return result
}

type tagView struct {
	Tag, Value string
}

func elementViews(data []string) []*elementView {
	result := make([]*elementView, 0, len(data))
	for i, v := range data {
		result = append(result, &elementView{ID: fmt.Sprintf("[%d]", i), Value: v})
	}
	return result
}

type elementView struct {
	ID, Value string
}
