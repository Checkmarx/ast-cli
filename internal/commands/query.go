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
	failedListingRepos             = "failed listing queries repositories"
	failedDeletingRepo             = "failed deleting queries repository"
	failedActivatingRepo           = "failed activating queries repository"
	failedActivatingAfterUploading = "failed activating queries repository after uploading it"
	failedCloningRepo              = "failed downloading queries repository"
	failedUploadingRepo            = "failed importing queries repository"
)

const QueriesRepoDestFileName = "queries-repository.tar.gz"

type QueryRepoView struct {
	Name         string
	IsActive     string    `format:"name:Is active"`
	LastModified time.Time `format:"name:Last modified;time:06-01-02 15:04:05"`
}

func NewQueryCommand(queryWrapper wrappers.QueriesWrapper, uploadsWrapper wrappers.UploadsWrapper) *cobra.Command {
	queryCmd := &cobra.Command{
		Use:   "query",
		Short: "Manage queries",
	}
	downloadCmd := &cobra.Command{
		Use:   "download [name] (default is the active repository)",
		Short: "Download a remote query repository to a local archive file",
		RunE:  runDownload(queryWrapper),
	}
	uploadCmd := &cobra.Command{
		Use:   "upload <repository>",
		Short: "Upload local query repository archive file (tarball format) to AST",
		RunE:  runUpload(queryWrapper, uploadsWrapper),
	}
	uploadCmd.PersistentFlags().StringP(queriesRepoNameFlag, queriesRepoNameSh, "",
		"A override name for your custom queries repository (default is the repository file name)")
	uploadCmd.PersistentFlags().BoolP(queriesRepoActivateFlag, queriesRepoActivateSh, false,
		"Whether to activate repository after uploading")
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List query repositories",
		RunE:  runList(queryWrapper),
	}
	activateCmd := &cobra.Command{
		Use:   "activate <name>",
		Short: "Activate a queries repository for the engine usage",
		RunE:  runActivate(queryWrapper),
	}
	deleteCmd := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a query repository",
		RunE:  runDelete(queryWrapper),
	}
	queryCmd.AddCommand(downloadCmd, uploadCmd, listCmd, activateCmd, deleteCmd)
	return queryCmd
}

func runDownload(queryWrapper wrappers.QueriesWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var name string
		if len(args) > 0 {
			name = args[0]
		}

		repo, errorModel, err := queryWrapper.Download(name)
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

		destFile := filepath.Join(pwdDir, QueriesRepoDestFileName)
		destWriter, err := os.Create(destFile)
		if err != nil {
			return errors.Wrapf(err, "%s failed creating file to download into it", failedCloningRepo)
		}

		defer destWriter.Close()
		_, _ = cmd.OutOrStdout().Write([]byte("Cloning into " + destFile + "\n"))
		_, err = io.Copy(destWriter, repo)
		if err != nil {
			return errors.Wrap(err, failedCloningRepo)
		}

		return nil
	}
}

func runUpload(queryWrapper wrappers.QueriesWrapper, uploadsWrapper wrappers.UploadsWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.Errorf("%s: Please provide a path to queries repository", failedUploadingRepo)
		}

		repoFile := args[0]
		nameOverride, _ := cmd.Flags().GetString(queriesRepoNameFlag)
		toActivate, _ := cmd.Flags().GetBool(queriesRepoActivateFlag)
		preSignedURL, err := uploadsWrapper.UploadFile(repoFile)
		if err != nil {
			return errors.Wrapf(err, "%s: failed to upload repository file", failedUploadingRepo)
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
			return errors.Wrap(err, failedUploadingRepo)
		}

		if errorModel != nil {
			return errors.Errorf("%s: CODE: %d, %s", failedUploadingRepo, errorModel.Code, errorModel.Message)
		}

		if toActivate {
			errorModel, err = queryWrapper.Activate(nameOverride)
			if err != nil {
				return errors.Wrap(err, failedActivatingAfterUploading)
			}

			if errorModel != nil {
				return errors.Errorf("%s: CODE: %d, %s", failedActivatingAfterUploading,
					errorModel.Code, errorModel.Message)
			}
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
			return errors.Errorf("%s: Please provide a queries repository name", failedActivatingRepo)
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
			return errors.Errorf("%s: Please provide a queries repository name", failedDeletingRepo)
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

func toQueryRepoViews(models []*queriesobjects.QueriesRepo) []*QueryRepoView {
	result := make([]*QueryRepoView, len(models))
	for i, model := range models {
		result[i] = toQueryRepoView(model)
	}

	return result
}

func toQueryRepoView(model *queriesobjects.QueriesRepo) *QueryRepoView {
	return &QueryRepoView{
		Name:         model.Name,
		LastModified: model.LastModified,
		IsActive: func() string {
			if model.IsActive {
				return "active"
			}

			return "inactive"
		}(),
	}
}
