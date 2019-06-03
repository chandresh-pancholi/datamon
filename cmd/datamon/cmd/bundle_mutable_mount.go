// Copyright © 2018 One Concern

package cmd

import (
	"context"
	"fmt"

	daemonizer "github.com/jacobsa/daemonize"

	"github.com/oneconcern/datamon/pkg/core"
	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage/gcs"
	"github.com/oneconcern/datamon/pkg/storage/localfs"
	"github.com/spf13/afero"

	"github.com/spf13/cobra"
)

// Mount a mutable view of a bundle
var mutableMountBundleCmd = &cobra.Command{
	Use:   "new",
	Short: "Create a bundle incrementally with filesystem operations",
	Long:  "Write directories and files to the mountpoint.  Unmount or send SIGINT to this process to save.",
	Run: func(cmd *cobra.Command, args []string) {
		if repoParams.ContributorEmail == "" {
			logFatalln(fmt.Errorf("contributor email must be set in config or as a cli param"))
		}
		if repoParams.ContributorName == "" {
			logFatalln(fmt.Errorf("contributor name must be set in config or as a cli param"))
		}

		if bundleOptions.Daemonize {
			runDaemonized()
			return
		}

		DieIfNotDirectory(bundleOptions.DataPath)

		metadataSource, err := gcs.New(repoParams.MetadataBucket, config.Credential)
		if err != nil {
			onDaemonError(err)
		}
		blobStore, err := gcs.New(repoParams.BlobBucket, config.Credential)
		if err != nil {
			onDaemonError(err)
		}
		consumableStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), bundleOptions.DataPath))

		bd := core.NewBDescriptor(
			core.Message(bundleOptions.Message),
			core.Contributors([]model.Contributor{{
				Name:  repoParams.ContributorName,
				Email: repoParams.ContributorEmail,
			},
			}),
		)
		bundle := core.New(bd,
			core.Repo(repoParams.RepoName),
			core.BlobStore(blobStore),
			core.ConsumableStore(consumableStore),
			core.MetaStore(metadataSource),
		)

		fs, err := core.NewMutableFS(bundle, bundleOptions.DataPath)
		if err != nil {
			onDaemonError(err)
		}
		err = fs.MountMutable(bundleOptions.MountPath)
		if err != nil {
			onDaemonError(err)
		}

		registerSIGINTHandlerMount(bundleOptions.MountPath)
		if err = daemonizer.SignalOutcome(nil); err != nil {
			logFatalln(err)
		}
		if err = fs.JoinMount(context.Background()); err != nil {
			logFatalln(err)
		}
		if err = fs.Commit(); err != nil {
			logFatalln(err)
		}
		fmt.Printf("bundle: %v\n", bundle.BundleID)
	},
}

func init() {

	requiredFlags := []string{addRepoNameOptionFlag(mutableMountBundleCmd)}
	addBucketNameFlag(mutableMountBundleCmd)
	addDaemonizeFlag(mutableMountBundleCmd)
	addBlobBucket(mutableMountBundleCmd)
	requiredFlags = append(requiredFlags, addDataPathFlag(mutableMountBundleCmd))
	requiredFlags = append(requiredFlags, addMountPathFlag(mutableMountBundleCmd))
	requiredFlags = append(requiredFlags, addCommitMessageFlag(mutableMountBundleCmd))

	for _, flag := range requiredFlags {
		err := mutableMountBundleCmd.MarkFlagRequired(flag)
		if err != nil {
			logFatalln(err)
		}
	}

	mountBundleCmd.AddCommand(mutableMountBundleCmd)
}
