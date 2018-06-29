// Copyright © 2018 One Concern

package cmd

import (
	"log"
	"path/filepath"

	"github.com/oneconcern/trumpet"
	"github.com/spf13/cobra"
)

// addCmd represents the add command
var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a file to a bundle for commit",
	Long: `Add a file or group of files to a bundle for commit.

This command supports providing one or more glob patterns
`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		tpt, err := trumpet.New("")
		if err != nil {
			log.Fatalln(err)
		}

		repo, err := tpt.GetRepo("hello-there")
		if err != nil {
			log.Fatalln(err)
		}

		for _, arg := range args {
			pths, err := filepath.Glob(arg)
			if err != nil {
				log.Fatalln(err)
			}
			for _, pth := range pths {
				addBlob, err := trumpet.UnstagedFilePath(pth)
				if err != nil {
					log.Fatalln(err)
				}

				hash, isNew, err := repo.Stage().Add(addBlob)
				if err != nil {
					log.Fatalln(err)
				}

				log.Println("added file", hash, "is new:", isNew)
			}
		}
	},
}

func init() {
	bundleCmd.AddCommand(addCmd)
	for i := 1; i < 100; i++ {
		addCmd.MarkZshCompPositionalArgumentFile(i, "*")
	}
	addRepoNameOption(addCmd)
}
