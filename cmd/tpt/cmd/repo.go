// Copyright © 2018 One Concern

package cmd

import (
	"context"

	"github.com/oneconcern/trumpet/pkg/engine"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/spf13/cobra"
)

var repoOptions struct {
	Name        string
	Description string
}

// repoCmd represents the repo command
var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Data Repo management related operations",
	Long: `Data repository management related operations for trumpet.

Repositories don't carry much content until a commit is made.
`,
}

func init() {
	rootCmd.AddCommand(repoCmd)
}

func initContext() context.Context {
	sp := opentracing.StartSpan("entrypoint").SetTag("service", "tptcli")
	return opentracing.ContextWithSpan(context.Background(), sp)
}

func initNamedRepo(ctx context.Context) (*engine.Runtime, *engine.Repo, error) {
	tpt, err := engine.New(&opentracing.NoopTracer{}, "")
	if err != nil {
		return nil, nil, err
	}

	repo, err := tpt.GetRepo(ctx, repoOptions.Name)
	if err != nil {
		return nil, nil, err
	}
	return tpt, repo, nil
}

func addRepoOptions(cmd *cobra.Command) error {
	fls := cmd.Flags()
	if err := addRepoNameOption(cmd); err != nil {
		return err
	}
	fls.StringVar(&repoOptions.Description, "description", "", "A description of this repository")
	return nil
}

func addRepoNameOption(cmd *cobra.Command) error {
	fls := cmd.Flags()
	fls.StringVar(&repoOptions.Name, "name", "", "The name of this repository")
	return cmd.MarkFlagRequired("name")
}

func addRepoFlag(cmd *cobra.Command) error {
	cmd.Flags().StringVar(&repoOptions.Name, "repo", "", "The name of the repository this command applies to")
	return cmd.MarkFlagRequired("repo")
}

func addPersistentRepoFlag(cmd *cobra.Command) error {
	cmd.PersistentFlags().StringVar(&repoOptions.Name, "repo", "", "The name of the repository this command applies to")
	return cmd.MarkPersistentFlagRequired("repo")
}
