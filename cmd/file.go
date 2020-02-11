package cmd

import (
	"github.com/spf13/cobra"
)

var fileCmd = &cobra.Command{
	Use: "file",
	Run: func(cmd *cobra.Command, args []string) {
		// x.DefaultClient()
		// client := auth.GetAccessClient()
		// resp, err := client.Get(x.GetApiUrlPath("file"))
		// if err != nil {
		// 	log.Fatal(err)
		// }
		// b, err := ioutil.ReadAll(resp.Body)
		// if err != nil {
		// 	log.Fatal(err)
		// }
		// fmt.Println(string(b))
	},
}

func init() {
	rootCmd.AddCommand(fileCmd)
}
