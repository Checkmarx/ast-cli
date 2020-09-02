package commands

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/checkmarxDev/sast-queries/pkg/v1/queriesobjects"

	"github.com/checkmarxDev/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	failedListingRepos   = "failed listing queries repos"
	failedDeletingRepo   = "failed deleting queries repo"
	failedActivatingRepo = "failed activating queries repo"
	failedCloningRepo    = "failed cloning queries repo"
	failedImportingRepo  = "failed importing queries repo"
)

const queriesRepoDistFileName = "queries-repo.tar.gz"

type queryRepoView struct {
	Name         string
	IsActive     bool      `format:"name:Is active"`
	LastModified time.Time `format:"name:Last modified;time:06-01-02 15:04:05"`
}

func NewQueryCommand(queryWrapper wrappers.QueriesWrapper, uploadsWrapper wrappers.UploadsWrapper) *cobra.Command {
	queryCmd := &cobra.Command{
		Use:   "query",
		Short: "Manage queries",
	}
	cloneCmd := &cobra.Command{
		Use:   "clone [name]",
		Short: "Clone queries repo tarball into current directory (blank name for the active queries repo)",
		RunE:  runClone(queryWrapper),
	}
	importCmd := &cobra.Command{
		Use:   "import",
		Short: "Import your custom queries repo into ast",
		RunE:  runImport(queryWrapper, uploadsWrapper),
	}
	importCmd.PersistentFlags().StringP(queriesRepoFileFlag, queriesRepoFileFlagSh, "",
		"File path to custom queries repo tarball")
	importCmd.PersistentFlags().StringP(queriesRepoNameFlag, queriesRepoNameSh, "",
		"A override name for your custom queries repo (default is the repo file name)")
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all the exists queries repos",
		RunE:  runList(queryWrapper),
	}
	activateCmd := &cobra.Command{
		Use:   "activate <name>",
		Short: "Activate exists queries repo for the engine usage",
		RunE:  runActivate(queryWrapper),
	}
	deleteCmd := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete exists queries repo",
		RunE:  runDelete(queryWrapper),
	}
	queryCmd.AddCommand(cloneCmd, importCmd, listCmd, activateCmd, deleteCmd)
	return queryCmd
}

func runClone(queryWrapper wrappers.QueriesWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var name string
		if len(args) > 0 {
			name = args[0]
		}

		repo, errorModel, err := queryWrapper.Clone(name)
		if err != nil {
			return errors.Wrap(err, failedCloningRepo)
		}

		if errorModel != nil {
			return errors.Errorf("%s: CODE: %d, %s", failedCloningRepo, errorModel.Code, errorModel.Message)
		}

		defer repo.Close()
		pwdDir, err := os.Getwd()
		if err != nil {
			return errors.Wrapf(err, "%s: failed get current directory path", failedCloningRepo)
		}

		distFile := filepath.Join(pwdDir, queriesRepoDistFileName)
		distWriter, err := os.Create(distFile)
		if err != nil {
			return errors.Wrapf(err, "%s failed creating file to clone into", failedCloningRepo)
		}

		defer distWriter.Close()
		_, _ = cmd.OutOrStdout().Write([]byte("Cloning into " + distFile + "\n"))
		_, err = io.Copy(distWriter, repo)
		if err != nil {
			return errors.Wrap(err, failedCloningRepo)
		}

		return nil
	}
}

func runImport(queryWrapper wrappers.QueriesWrapper, uploadsWrapper wrappers.UploadsWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		repoFile, _ := cmd.Flags().GetString(queriesRepoFileFlag)
		nameOverride, _ := cmd.Flags().GetString(queriesRepoNameFlag)

		if repoFile == "" {
			return errors.Errorf("%s: Please provide a tarball repo file path", failedImportingRepo)
		}

		preSignedURL, err := uploadsWrapper.UploadFile(repoFile)
		if err != nil {
			return errors.Wrapf(err, "%s: failed to upload repo file", failedImportingRepo)
		}

		PrintIfVerbose(fmt.Sprintf("Uploading file to %s\n", *preSignedURL))
		// Default to the given repo file name
		if nameOverride == "" {
			baseFileName := filepath.Base(repoFile)
			// strip the file extension
			nameOverride = strings.TrimSuffix(baseFileName, filepath.Ext(baseFileName))
		}

		errorModel, err := queryWrapper.Import(*preSignedURL, nameOverride)
		if err != nil {
			return errors.Wrap(err, failedImportingRepo)
		}

		if errorModel != nil {
			return errors.Errorf("%s: CODE: %d, %s", failedActivatingRepo, errorModel.Code, errorModel.Message)
		}

		return nil
	}
}

func runList(queryWrapper wrappers.QueriesWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		queriesRepos, err := queryWrapper.List()
		if err != nil {
			return errors.Wrap(err, failedListingRepos)
		}

		err = Print(cmd.OutOrStdout(), toQueryRepoViews(queriesRepos))
		if err != nil {
			return errors.Wrap(err, failedListingRepos)
		}

		return nil
	}
}

func runActivate(queryWrapper wrappers.QueriesWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.Errorf("%s: Please provide a queries repo name", failedActivatingRepo)
		}

		name := args[0]
		errorModel, err := queryWrapper.Activate(name)
		if err != nil {
			return errors.Wrap(err, failedActivatingRepo)
		}

		if errorModel != nil {
			return errors.Errorf("%s: CODE: %d, %s", failedActivatingRepo, errorModel.Code, errorModel.Message)
		}

		return nil
	}
}

func runDelete(queryWrapper wrappers.QueriesWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.Errorf("%s: Please provide a queries repo name", failedDeletingRepo)
		}

		name := args[0]
		errorModel, err := queryWrapper.Delete(name)
		if err != nil {
			return errors.Wrap(err, failedDeletingRepo)
		}

		if errorModel != nil {
			return errors.Errorf("%s: CODE: %d, %s", failedDeletingRepo, errorModel.Code, errorModel.Message)
		}

		return nil
	}
}

func toQueryRepoViews(models []*queriesobjects.QueriesRepo) []*queryRepoView {
	result := make([]*queryRepoView, len(models))
	for i, model := range models {
		result[i] = toQueryRepoView(model)
	}

	return result
}

func toQueryRepoView(model *queriesobjects.QueriesRepo) *queryRepoView {
	return &queryRepoView{
		Name:         model.Name,
		LastModified: model.LastModified,
		IsActive:     model.IsActive,
	}
}
