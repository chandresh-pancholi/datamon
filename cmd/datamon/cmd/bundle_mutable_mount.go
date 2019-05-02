// Copyright © 2018 One Concern

package cmd

import (
	"log"
	"os"
	"os/signal"

	"github.com/oneconcern/datamon/pkg/core"
	"github.com/oneconcern/datamon/pkg/storage/gcs"
	"github.com/oneconcern/datamon/pkg/storage/localfs"
	"github.com/spf13/afero"

	"github.com/spf13/cobra"
)

// Mount a read only view of a bundle
var mutableMountBundleCmd = &cobra.Command{
	Use:   "mutable-mount",
	Short: "Create a bundle incrementally with filesystem operations",
	Long:  "Write directories and files to the mountpoint.  Unmount to discard or send SIGINT to this process to save.",
	Run: func(cmd *cobra.Command, args []string) {

		DieIfNotDirectory(bundleOptions.DataPath)

		metadataSource, err := gcs.New(repoParams.MetadataBucket, config.Credential)
		if err != nil {
			logFatalln(err)
		}
		blobStore, err := gcs.New(repoParams.BlobBucket, config.Credential)
		if err != nil {
			logFatalln(err)
		}
		consumableStore := localfs.New(afero.NewBasePathFs(afero.NewOsFs(), bundleOptions.DataPath))

		bd := core.NewBDescriptor()
		bundle := core.New(bd,
			core.Repo(repoParams.RepoName),
			core.BlobStore(blobStore),
			core.ConsumableStore(consumableStore),
			core.MetaStore(metadataSource),
		)

		fs, err := core.NewMutableFS(bundle, bundleOptions.DataPath)
		if err != nil {
			logFatalln(err)
		}
		err = fs.MountMutable(bundleOptions.MountPath)
		if err != nil {
			logFatalln(err)
		}

		signalChan := make(chan os.Signal, 1)
		signal.Notify(signalChan, os.Interrupt)
		
		<-signalChan

		log.Println("caught interrupt signal")

		err = fs.Unmount(bundleOptions.MountPath)
		if err != nil {
			logFatalln(err)
		}

		log.Printf("bundle: %v\n", bundle.BundleID)
	},
}

func init() {

	requiredFlags := []string{addRepoNameOptionFlag(mutableMountBundleCmd)}
	addBucketNameFlag(mutableMountBundleCmd)
	addBlobBucket(mutableMountBundleCmd)
	requiredFlags = append(requiredFlags, addDataPathFlag(mutableMountBundleCmd))
	requiredFlags = append(requiredFlags, addMountPathFlag(mutableMountBundleCmd))

	for _, flag := range requiredFlags {
		err := mutableMountBundleCmd.MarkFlagRequired(flag)
		if err != nil {
			logFatalln(err)
		}
	}

	bundleCmd.AddCommand(mutableMountBundleCmd)
}
