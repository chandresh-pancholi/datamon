package cmd

import (
	"bytes"
	"context"
	"fmt"
	"log"

	"github.com/oneconcern/datamon/pkg/core"
	"github.com/oneconcern/datamon/pkg/model"

	"github.com/spf13/cobra"
)

func applyBundleTemplate(bundle model.BundleDescriptor) error {
	var buf bytes.Buffer
	err := bundleDescriptorTemplate.Execute(&buf, bundle)
	if err != nil {
		return fmt.Errorf("executing template: %w", err)
	}
	log.Println(buf.String())
	return nil
}

// BundleListCommand describes the CLI command for listing bundles
var BundleListCommand = &cobra.Command{
	Use:   "list",
	Short: "List bundles",
	Long:  "List the bundles in a repo, ordered by their bundle ID",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		remoteStores, err := paramsToRemoteCmdStores(ctx, params)
		if err != nil {
			logFatalln(err)
		}
		err = core.ListBundlesApply(params.repo.RepoName, remoteStores.meta, applyBundleTemplate,
			core.ConcurrentBundleList(params.core.ConcurrencyFactor),
			core.BundleBatchSize(params.core.BatchSize))
		if err != nil {
			logFatalln(err)
		}
	},
}

func init() {

	requiredFlags := []string{addRepoNameOptionFlag(BundleListCommand)}

	addBucketNameFlag(BundleListCommand)
	addBlobBucket(BundleListCommand)
	addCoreConcurrencyFactorFlag(BundleListCommand)
	addBatchSizeFlag(BundleListCommand)

	for _, flag := range requiredFlags {
		err := BundleListCommand.MarkFlagRequired(flag)
		if err != nil {
			logFatalln(err)
		}
	}

	bundleCmd.AddCommand(BundleListCommand)
}
