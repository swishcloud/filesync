package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/swishcloud/filesync/client"
)

var syncCmd = &cobra.Command{
	Use: "sync",
	Run: func(cmd *cobra.Command, args []string) {
		file_path, err := cmd.Flags().GetString("file_path")
		if err != nil {
			log.Println(err)
		}
		err = client.SendFile(file_path, skip_tls_verify)
		if err != nil {
			fmt.Println(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)
	syncCmd.Flags().String("file_path", "", "the path of file to upload")
	syncCmd.MarkFlagRequired("file_path")
}
