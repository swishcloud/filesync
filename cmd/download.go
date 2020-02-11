package cmd

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/swishcloud/filesync/client"
)

var downloadCmd = &cobra.Command{
	Use: "download",
	Run: func(cmd *cobra.Command, args []string) {
		file_id, err := cmd.Flags().GetString("file_id")
		if err != nil {
			log.Println(err)
		}
		client.DownloadFile(file_id, skip_tls_verify)
	},
}

func init() {
	rootCmd.AddCommand(downloadCmd)
	downloadCmd.Flags().String("file_id", "", "Id of file to download")
	downloadCmd.MarkFlagRequired("file_id")
}
