package cmd

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log"
	"strings"

	"github.com/swishcloud/gostudy/keygenerator"

	"github.com/spf13/cobra"
	"github.com/swishcloud/filesync/internal"

	"golang.org/x/oauth2"
)

var loginCmd = &cobra.Command{
	Use: "login",
	Run: func(cmd *cobra.Command, args []string) {
		token_path, err := cmd.Flags().GetString("token_path")
		if err != nil {
			panic(err)
		}
		internal.Token_save_path(token_path)
		_, err = internal.GetToken()
		if err == nil {
			return
		}

		pkce, err := keygenerator.NewKey(43, false, false, false, true)
		if err != nil {
			panic(err)
		}
		conf := internal.OAuth2Config()
		sha256_hased_pkce := sha256.Sum256([]byte(pkce))
		encoded_pcke := base64.StdEncoding.EncodeToString(sha256_hased_pkce[:])
		encoded_pcke = strings.Replace(encoded_pcke, "=", "", -1)
		encoded_pcke = strings.Replace(encoded_pcke, "+", "-", -1)
		encoded_pcke = strings.Replace(encoded_pcke, "/", "_", -1)
		stateStr, err := keygenerator.NewKey(43, false, false, false, true)
		if err != nil {
			panic(err)
		}
		url := conf.AuthCodeURL(stateStr, oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("code_challenge", encoded_pcke), oauth2.SetAuthURLParam("code_challenge_method", "S256"))
		fmt.Println("copy this url then open in browser:", url)
		fmt.Print("Enter authenfication code:")
		code := ""
		fmt.Scan(&code)
		token, err := conf.Exchange(context.WithValue(context.Background(), oauth2.HTTPClient, internal.HttpClient()), code, oauth2.SetAuthURLParam("code_verifier", pkce))
		if err != nil {
			log.Fatal(err)
		}
		internal.SaveToken(token)
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
	loginCmd.Flags().String("token_path", "", "the path to read or write token file")
	loginCmd.MarkFlagRequired("token_path")
}
