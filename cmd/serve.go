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
		port, err := cmd.Flags().GetString("port")
		if err != nil {
			log.Println(err)
		}
		root, err := cmd.Flags().GetString("root")
		if err != nil {
			log.Println(err)
		}

		repeat, err := cmd.Flags().GetString("repeat")
		if err != nil {
			log.Println(err)
		}

		filters, err := cmd.Flags().GetString("filters")
		if err != nil {
			log.Println(err)
		}
		s := server.NewFileSyncServer(port, root, repeat, filters)
		s.Serve()
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().String("port", "2000", "server listen port")
	serveCmd.Flags().String("root", "", "file location root path")
	serveCmd.Flags().String("repeat", "", "repeat file from another server addr")
	serveCmd.Flags().String("filters", "", "ignore files with specified name while repeating files,use colon to separate multiple names")
	serveCmd.MarkFlagRequired("root")
}
