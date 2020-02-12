package cmd

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/swishcloud/filesync/server"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "start serving",
	Run: func(cmd *cobra.Command, args []string) {
		config, err := cmd.Flags().GetString("config")
		if err != nil {
			log.Println(err)
		}
		s := server.NewFileSyncServer(config, skip_tls_verify)
		s.Serve()
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().String("config", "server.yaml", "the config file path of this server")
	serveCmd.MarkFlagRequired("config")
}
