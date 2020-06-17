package internal

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/swishcloud/filesync/storage/models"
	"github.com/swishcloud/gostudy/common"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v2"
)

type globalConfig struct {
	BaseApiUrlPath   string
	ExchangeTokenUrl string
	AuthCodeUrl      string
	AuthUrl          string
	TokenURL         string
}

const TokenHeaderKey = "access_token"

const token_save_path = ".cache/token"

var httpClient *http.Client

func Initialize(c *http.Client) {
	httpClient = c
}
func HttpClient() *http.Client {
	return httpClient
}

var gc *globalConfig

func GlobalConfig() globalConfig {
	if gc == nil {
		gc = &globalConfig{}
		if os.Getenv("development") != "true" {
			gc.BaseApiUrlPath = "https://192.168.100.8:2002/api/"
			gc.ExchangeTokenUrl = "https://192.168.100.8:2002/api/exchange_token"
			gc.AuthCodeUrl = "https://192.168.100.8:2002/api/auth_code_url"
			gc.AuthUrl = "https://localhost:8010/oauth2/auth"
			gc.TokenURL = "https://localhost:8010/oauth2/token"
		} else {
			gc.BaseApiUrlPath = "https://cloud.swish-cloud.com/api/"
			gc.ExchangeTokenUrl = "https://cloud.swish-cloud.com/api/exchange_token"
			gc.AuthCodeUrl = "https://cloud.swish-cloud.com/api/auth_code_url"
		}
	}
	return *gc
}
func OAuth2Config() *oauth2.Config {
	conf := oauth2.Config{}
	conf.ClientID = "FILESYNC_MOBILE"
	conf.Scopes = []string{"offline"}
	conf.Endpoint = oauth2.Endpoint{
		AuthURL:  GlobalConfig().AuthUrl,
		TokenURL: GlobalConfig().TokenURL,
	}
	return &conf
}
func GetLogs(start int64) ([]models.Log, error) {
	params := url.Values{}
	params.Add("start", strconv.FormatInt(start, 10))
	url := GlobalConfig().BaseApiUrlPath + "log" + "?" + params.Encode()
	token, err := GetToken()
	if err != nil {
		return nil, err
	}
	rar := common.NewRestApiRequest("GET", url, nil).SetAuthHeader(token)
	resp, err := rac.Do(rar)
	if err != nil {
		return nil, err
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	result := struct{ Data []models.Log }{}
	err = json.Unmarshal(b, &result)
	if err != nil {
		return nil, err
	}
	return result.Data, nil
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
func DeleteFile(file_id string) error {
	params := url.Values{}
	params.Add("file_id", file_id)
	token, err := GetToken()
	if err != nil {
		return err
	}
	url := GlobalConfig().BaseApiUrlPath + "file" + "?" + params.Encode()
	rar := common.NewRestApiRequest("DELETE", url, nil).SetAuthHeader(token)
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
func GetDirectory(p_id string, name string, skip_tls_verify bool) (map[string]interface{}, error) {
	params := url.Values{}
	params.Add("p_id", p_id)
	params.Add("name", name)
	token, err := GetToken()
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
	token, err := GetToken()
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

func GetToken() (*oauth2.Token, error) {
	b, err := ioutil.ReadFile(token_save_path)
	if err != nil {
		return nil, err
	}
	token := &oauth2.Token{}
	if err := yaml.Unmarshal(b, token); err != nil {
		return nil, err
	}
	ts := OAuth2Config().TokenSource(context.WithValue(context.Background(), "", HttpClient()), token)
	t, err := ts.Token()
	if err != nil {
		return nil, err
	}
	if t.AccessToken != token.AccessToken {
		log.Println("got refreshed new token")
		SaveToken(t)
		return t, nil
	}
	return token, nil
}

func SaveToken(token *oauth2.Token) {
	if b, err := yaml.Marshal(token); err != nil {
		panic(err)
	} else if err := ioutil.WriteFile(token_save_path, b, os.ModePerm); err != nil {
		panic(err)
	}
}
