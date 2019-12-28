package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "filesync",
	Short: "filesync is file transfering server",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("welcome to filesync")
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
