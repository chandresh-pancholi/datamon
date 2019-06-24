package cmd

import (
	"fmt"
	"time"

	"github.com/oneconcern/datamon/pkg/storage/gcs"

	"github.com/oneconcern/datamon/pkg/core"

	"github.com/oneconcern/datamon/pkg/model"

	"github.com/spf13/cobra"
)

var repoCreate = &cobra.Command{
	Use:   "create",
	Short: "Create a named repo",
	Long: "Create a repo. Repo names must not contain special characters. " +
		"Allowed characters Unicode characters, digits and hyphen. Example: dm-test-repo-1",
	Run: func(cmd *cobra.Command, args []string) {
		if params.repo.ContributorEmail == "" {
			logFatalln(fmt.Errorf("contributor email must be set in config or as a cli param"))
		}
		if params.repo.ContributorName == "" {
			logFatalln(fmt.Errorf("contributor name must be set in config or as a cli param"))
		}
		store, err := gcs.New(params.repo.MetadataBucket, config.Credential)
		if err != nil {
			logFatalln(err)
		}

		repo := model.RepoDescriptor{
			Name:        params.repo.RepoName,
			Description: params.repo.Description,
			Timestamp:   time.Now(),
			Contributor: model.Contributor{
				Email: params.repo.ContributorEmail,
				Name:  params.repo.ContributorName,
			},
		}
		err = core.CreateRepo(repo, store)
		if err != nil {
			logFatalln(err)
		}
	},
}

func init() {

	// Metadata bucket
	requiredFlags := []string{addRepoNameOptionFlag(repoCreate)}
	// Description
	requiredFlags = append(requiredFlags, addRepoDescription(repoCreate))

	addContributorEmail(repoCreate)
	addContributorName(repoCreate)
	addBucketNameFlag(repoCreate)

	for _, flag := range requiredFlags {
		err := repoCreate.MarkFlagRequired(flag)
		if err != nil {
			logFatalln(err)
		}
	}

	repoCmd.AddCommand(repoCreate)
}
