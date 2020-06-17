package cmd

import (
	"fmt"
	"log"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/swishcloud/filesync/internal"
	"github.com/swishcloud/filesync/x"

	"github.com/swishcloud/filesync/client"
	"github.com/swishcloud/gostudy/common"

	"github.com/spf13/cobra"
)

var uploadCmd = &cobra.Command{
	Use: "upload",
	Run: func(cmd *cobra.Command, args []string) {
		file_path, err := cmd.Flags().GetString("file_path")
		file_path, err = filepath.Abs(file_path)
		if err != nil {
			log.Fatal(err)
		}
		if err != nil {
			log.Fatal(err)
		}
		file_path = strings.ReplaceAll(file_path, "\\", "/")
		file_path = strings.TrimSuffix(file_path, "/")
		p_file_path := regexp.MustCompile(".*/").FindString(file_path)
		items := []*common.FileInfoWrapper{}
		err = common.ReadAllFiles(file_path, &items)
		if err != nil {
			log.Fatal(err)
		}
		failureNum := 0
		for index, item := range items {
			location := strings.Replace(item.Path, p_file_path, "", 1)
			location = regexp.MustCompile(".*/").FindString(location)
			location = strings.TrimSuffix(location, "/")
			fmt.Println("target location:", location)
			is_hidden, err := x.IsHidden(item.Path)
			if err != nil {
				log.Printf(err.Error())
				failureNum++
			} else {
				if item.Fi.IsDir() {
					fmt.Printf("found folder '%s'\r\n", item.Path)
					//ensure directory already created
					if err := internal.CreateDirectory(location, item.Fi.Name(), is_hidden); err != nil {
						log.Fatal(err.Error())
						failureNum++
					}
				} else {
					fmt.Printf("uploading file '%s'\r\n", item.Path)
					err = client.SendFile(item.Path, location, is_hidden)
					if err != nil {
						log.Printf(err.Error())
						failureNum++
					}
				}
			}
			fmt.Printf("progress: %d/%d failure:%d\r\n", index+1, len(items), failureNum)
		}
	},
}

func init() {
	rootCmd.AddCommand(uploadCmd)
	uploadCmd.Flags().String("file_path", "", "the path of file to upload")
	uploadCmd.MarkFlagRequired("file_path")
}
