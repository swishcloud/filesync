package cmd

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/swishcloud/gostudy/keygenerator"

	"github.com/spf13/cobra"
	"github.com/swishcloud/filesync/internal"

	"golang.org/x/oauth2"
)

func save_token_for_cpp(token *oauth2.Token) {
	if err := ioutil.WriteFile("token", []byte(token.AccessToken), os.ModePerm); err != nil {
		panic(err)
	}
}

var loginCmd = &cobra.Command{
	Use: "login",
	Run: func(cmd *cobra.Command, args []string) {
		token, err := internal.GetToken()
		if err == nil {
			save_token_for_cpp(token)
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
		url := conf.AuthCodeURL("state-string", oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("code_challenge", encoded_pcke), oauth2.SetAuthURLParam("code_challenge_method", "S256"))
		fmt.Println("copy this url then open in browser:", url)
		fmt.Print("Enter authenfication code:")
		code := ""
		fmt.Scan(&code)
		token, err = conf.Exchange(context.WithValue(context.Background(), "", internal.HttpClient()), code, oauth2.SetAuthURLParam("code_verifier", pkce))
		if err != nil {
			log.Fatal(err)
		}
		internal.SaveToken(token)
		save_token_for_cpp(token)
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
}
