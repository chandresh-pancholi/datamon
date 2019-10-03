package cmd

import (
	"bytes"
	"context"
	"log"
	"text/template"

	"github.com/oneconcern/datamon/pkg/core"

	"github.com/spf13/cobra"
)

var BundleListCommand = &cobra.Command{
	Use:   "list",
	Short: "List bundles",
	Long:  "List the bundles in a repo",
	Run: func(cmd *cobra.Command, args []string) {
		const listLineTemplateString = `{{.ID}} , {{.Timestamp}} , {{.Message}}`
		ctx := context.Background()
		listLineTemplate := template.Must(template.New("list line").Parse(listLineTemplateString))
		remoteStores, err := paramsToRemoteCmdStores(ctx, params)
		if err != nil {
			logFatalln(err)
		}
		bundleDescriptors, err := core.ListBundles(params.repo.RepoName, remoteStores.meta, core.ConcurrentBundleList(params.core.ConcurrencyFactor))
		if err != nil {
			logFatalln(err)
		}
		for _, bd := range bundleDescriptors {
			var buf bytes.Buffer
			err := listLineTemplate.Execute(&buf, bd)
			if err != nil {
				log.Println("executing template:", err)
			}
			log.Println(buf.String())
		}
	},
}

func init() {

	requiredFlags := []string{addRepoNameOptionFlag(BundleListCommand)}

	addBucketNameFlag(BundleListCommand)
	addBlobBucket(BundleListCommand)
	addCoreConcurrencyFactorFlag(BundleListCommand)

	for _, flag := range requiredFlags {
		err := BundleListCommand.MarkFlagRequired(flag)
		if err != nil {
			logFatalln(err)
		}
	}

	bundleCmd.AddCommand(BundleListCommand)
}
