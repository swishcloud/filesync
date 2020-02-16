package cmd

import (
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/swishcloud/filesync/auth"
	"github.com/swishcloud/filesync/internal"

	"github.com/spf13/cobra"
	"github.com/swishcloud/gostudy/common"
)

var loginCmd = &cobra.Command{
	Use: "login",
	Run: func(cmd *cobra.Command, args []string) {
		rar := common.NewRestApiRequest("GET", internal.GlobalConfig().AuthCodeUrl+"?state="+uuid.New().String(), nil)
		resp, err := internal.RestApiClient().Do(rar)
		if err != nil {
			log.Fatal(err)
		}
		m, err := common.ReadAsMap(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		url := m["data"].(string)
		fmt.Println("copy this url then open in browser:", url)
		fmt.Print("Enter authenfication code:")
		code := ""
		fmt.Scan(&code)
		rar = common.NewRestApiRequest("POST", internal.GlobalConfig().ExchangeTokenUrl, []byte("code="+code))
		resp, err = internal.RestApiClient().Do(rar)
		if err != nil {
			panic(err)
		}
		m, err = common.ReadAsMap(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		token := m["data"].(string)
		log.Println("exchanged token:", token)
		auth.SaveToken(token)

	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
}
