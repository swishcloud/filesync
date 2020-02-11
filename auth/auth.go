package auth

import (
	"io/ioutil"
	"os"

	"golang.org/x/oauth2"
)

const token_save_path = ".cache/token"

func GetToken() *oauth2.Token {
	b, err := ioutil.ReadFile(token_save_path)
	if err != nil {
		panic(err)
	}
	tokenstr := string(b)
	return &oauth2.Token{AccessToken: tokenstr}
}

func SaveToken(tokenstr string) {
	err := ioutil.WriteFile(token_save_path, []byte(tokenstr), os.ModePerm)
	if err != nil {
		panic(err)
	}
}
