package internal

import "os"

type globalConfig struct {
	BaseApiUrlPath   string
	ExchangeTokenUrl string
	AuthCodeUrl      string
}

var gc *globalConfig

func GlobalConfig() *globalConfig {
	if gc == nil {
		gc = &globalConfig{}
		if os.Getenv("development") == "true" {
			gc.BaseApiUrlPath = "https://localhost:2002/api/"
			gc.ExchangeTokenUrl = "https://localhost:2002/api/exchange_token"
			gc.AuthCodeUrl = "https://localhost:2002/api/auth_code_url"
		} else {
			gc.BaseApiUrlPath = "https://cloud.swish-cloud.com/api/"
			gc.ExchangeTokenUrl = "https://cloud.swish-cloud.com/api/exchange_token"
			gc.AuthCodeUrl = "https://cloud.swish-cloud.com/api/auth_code_url"
		}
	}
	return gc
}
