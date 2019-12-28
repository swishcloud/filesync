package cmd

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/swishcloud/filesync/client"
)

var pingCmd = &cobra.Command{
	Use: "ping",
	Run: func(cmd *cobra.Command, args []string) {
		server, err := cmd.Flags().GetString("server")
		if err != nil {
			log.Println(err)
		}
		client.Ping(server)
	},
}

func init() {
	rootCmd.AddCommand(pingCmd)
	pingCmd.Flags().String("server", "", "target server to ping")
	pingCmd.MarkFlagRequired("server")
}
