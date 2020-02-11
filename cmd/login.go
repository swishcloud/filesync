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
		rac := common.NewRestApiClient("GET", internal.GlobalConfig().AuthCodeUrl+"?state="+uuid.New().String(), nil, skip_tls_verify)
		resp, err := rac.DoExpect200Status()
		if err != nil {
			panic(err)
		}
		url := common.ReadAsMap(resp.Body)["data"].(string)
		fmt.Println("copy this url then open in browser:", url)
		fmt.Print("Enter authenfication code:")
		code := ""
		fmt.Scan(&code)
		rac = common.NewRestApiClient("POST", internal.GlobalConfig().ExchangeTokenUrl, []byte("code="+code), skip_tls_verify)
		resp, err = rac.DoExpect200Status()
		if err != nil {
			panic(err)
		}
		token := common.ReadAsMap(resp.Body)["data"].(string)
		log.Println("exchanged token:", token)
		auth.SaveToken(token)

	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
}
