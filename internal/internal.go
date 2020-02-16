package internal

import (
	"errors"
	"net/url"
	"os"
	"strconv"

	"github.com/swishcloud/filesync/auth"
	"github.com/swishcloud/gostudy/common"
	"golang.org/x/oauth2"
)

type globalConfig struct {
	BaseApiUrlPath   string
	ExchangeTokenUrl string
	AuthCodeUrl      string
}

const TokenHeaderKey = "access_token"

var gc *globalConfig

func GlobalConfig() *globalConfig {
	if gc == nil {
		gc = &globalConfig{}
		if os.Getenv("development") == "true" {
			gc.BaseApiUrlPath = "https://192.168.100.8:2002/api/"
			gc.ExchangeTokenUrl = "https://192.168.100.8:2002/api/exchange_token"
			gc.AuthCodeUrl = "https://192.168.100.8:2002/api/auth_code_url"
		} else {
			gc.BaseApiUrlPath = "https://cloud.swish-cloud.com/api/"
			gc.ExchangeTokenUrl = "https://cloud.swish-cloud.com/api/exchange_token"
			gc.AuthCodeUrl = "https://cloud.swish-cloud.com/api/auth_code_url"
		}
	}
	return gc
}

func GetFileData(file_name, md5, directory_path string, is_hidden bool, token *oauth2.Token) (map[string]interface{}, error) {
	params := url.Values{}
	params.Add("md5", md5)
	params.Add("name", file_name)
	params.Add("directory_path", directory_path)
	params.Add("is_hidden", strconv.FormatBool(is_hidden))
	url := GlobalConfig().BaseApiUrlPath + "file" + "?" + params.Encode()
	rar := common.NewRestApiRequest("GET", url, nil).SetAuthHeader(token)
	resp, err := rac.Do(rar)
	if err != nil {
		return nil, err
	}
	m, err := common.ReadAsMap(resp.Body)
	if err != nil {
		return nil, err
	}
	if m["error"] != nil {
		return nil, errors.New(m["error"].(string))
	}
	if m["data"] == nil {
		return nil, nil
	}
	return m["data"].(map[string]interface{}), nil
}
func GetDirectory(p_id string, name string, skip_tls_verify bool) (map[string]interface{}, error) {
	params := url.Values{}
	params.Add("p_id", p_id)
	params.Add("name", name)
	token, err := auth.GetToken()
	if err != nil {
		return nil, err
	}
	url := GlobalConfig().BaseApiUrlPath + "directory" + "?" + params.Encode()
	rar := common.NewRestApiRequest("GET", url, nil).SetAuthHeader(token)
	resp, err := rac.Do(rar)
	if err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	m, err := common.ReadAsMap(resp.Body)
	if err != nil {
		return nil, err
	}
	if m["error"] != nil {
		return nil, errors.New(m["error"].(string))
	}
	if m["data"] == nil {
		return nil, nil
	}
	return m["data"].(map[string]interface{}), nil
}

func CreateDirectory(path string, name string, is_hidden bool) error {
	params := url.Values{}
	params.Add("path", path)
	params.Add("name", name)
	params.Add("is_hidden", strconv.FormatBool(is_hidden))
	url := GlobalConfig().BaseApiUrlPath + "directory"
	token, err := auth.GetToken()
	if err != nil {
		return err
	}
	rar := common.NewRestApiRequest("POST", url, []byte(params.Encode())).SetAuthHeader(token)
	resp, err := rac.Do(rar)
	if err != nil {
		return err
	}
	m, err := common.ReadAsMap(resp.Body)
	if err != nil {
		return err
	}
	if m["error"] != nil {
		return errors.New(m["error"].(string))
	}
	return nil
}

var rac *common.RestApiClient

func InitRAC(skip_tls_verify bool) error {
	if rac != nil {
		return errors.New("can't repeatedly call this method")
	}
	rac = common.NewRestApiClient(skip_tls_verify)
	return nil
}
func RestApiClient() *common.RestApiClient {
	return rac
}

func GetApiUrlPath(p string) string {
	return GlobalConfig().BaseApiUrlPath + p
}
