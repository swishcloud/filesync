package server

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/google/uuid"
	"github.com/swishcloud/filesync/internal"
	"github.com/swishcloud/filesync/storage"

	"golang.org/x/oauth2"
	"gopkg.in/yaml.v2"

	"github.com/swishcloud/filesync/message"
	"github.com/swishcloud/filesync/message/models"
	"github.com/swishcloud/filesync/session"
	"github.com/swishcloud/filesync/x"
	"github.com/swishcloud/gostudy/common"
)

type FileSyncServer struct {
	config          *config
	skip_tls_verify bool
	httpClient      *http.Client
	storage         *storage.SQLITEManager
	clients         []*client
	connect         chan *session.Session
	disconnect      chan *session.Session
}
type client struct {
	session *session.Session
	class   int
}
type config struct {
	IntrospectTokenURL string `yaml:"IntrospectTokenURL"`
	Port               string `yaml:"Port"`
	FileLocation       string `yaml:"FileLocation"`
}

func (cfg *config) blockDir() string {
	return cfg.FileLocation + "/block/"
}
func (cfg *config) fileDir() string {
	return cfg.FileLocation + "/file/"
}
func (cfg *config) tempDir() string {
	return cfg.FileLocation + "/tmp/"
}

func NewFileSyncServer(config_file_path string, skip_tls_verify bool) *FileSyncServer {
	s := &FileSyncServer{}
	s.storage = &storage.SQLITEManager{}
	s.clients = []*client{}
	s.connect = make(chan *session.Session)
	s.disconnect = make(chan *session.Session)
	s.storage.Initialize()
	s.skip_tls_verify = skip_tls_verify
	s.httpClient = &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: skip_tls_verify}}}
	http.DefaultClient = s.httpClient
	s.config = new(config)
	b, err := ioutil.ReadFile(config_file_path)
	if err != nil {
		log.Fatal(err)
	}
	err = yaml.Unmarshal(b, s.config)
	if err != nil {
		log.Fatal(err)
	}
	err = os.MkdirAll(s.config.blockDir(), os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}
	err = os.MkdirAll(s.config.fileDir(), os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}
	err = os.MkdirAll(s.config.tempDir(), os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}
	return s
}
func (s *FileSyncServer) Serve() {
	// Listen on TCP port 2000 on all available unicast and
	// anycast IP addresses of the local system.
	l, err := net.Listen("tcp", ":"+s.config.Port)
	log.Println("accepting tcp connections on port", s.config.Port)
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()
	// Handle the sessions in a new goroutine.
	go s.serveSessions()
	for {
		// Wait for a connection.
		conn, err := l.Accept()
		if err != nil {
			log.Fatal(err)
		}
		s.connect <- session.NewSession(conn)
	}
}
func checkFile(file models.File, fullPath string) {
	if file.IsHidden {
		err := x.HideFile(fullPath)
		if err != nil {
			panic(err)
		}
	}
}
func (s *FileSyncServer) checkToken(msg *message.Message) (user_id string, token *oauth2.Token, err error) {
	tokenstr := msg.Header[internal.TokenHeaderKey]
	if tokenstr == nil {
		return "", nil, errors.New("token is missing")
	}
	token = &oauth2.Token{AccessToken: tokenstr.(string)}
	rar := common.NewRestApiRequest("GET", s.config.IntrospectTokenURL, nil).SetAuthHeader(token)
	resp, err := internal.RestApiClient().Do(rar)
	if err != nil {
		return "", nil, err
	}
	m, err := common.ReadAsMap(resp.Body)
	if err != nil {
		return "", nil, err
	}
	if m["error"] != nil {
		return "", nil, errors.New(m["error"].(string))
	}
	data := m["data"].(map[string]interface{})
	isActive := data["active"].(bool)
	if !isActive {
		return "", nil, errors.New("the token is not valid")
	}
	sub := data["sub"].(string)
	return sub, token, nil
}
func (s *FileSyncServer) serveSessions() {
	for {
		select {
		case connect := <-s.connect:
			client := &client{session: connect, class: 1}
			s.clients = append(s.clients, client)
			go s.serveClient(client)
		case disconect := <-s.disconnect:
			disconect.Close()
			for index, item := range s.clients {
				if item.session == disconect {
					s.clients = append(s.clients[:index], s.clients[index+1:]...)
					break
				}
			}
		}
	}
}
func (s *FileSyncServer) serveClient(client *client) {
	session := client.session
	defer func() {
		if err := recover(); err != nil {
			log.Println("disconnect session:", session, "cause:", err)
			s.disconnect <- session
		}
	}()
	log.Println("New session:", session)
	for {
		msg, err := session.ReadMessage()
		if err != nil {
			panic(err)
		}
		switch msg.MsgType {
		case message.MT_PING:
			reply_msg := new(message.Message)
			reply_msg.MsgType = message.MT_PANG
			session.SendMessage(reply_msg, nil, 0)
		case message.MT_FILE:
			// _, token, err := s.checkToken(msg)
			// if err != nil {
			// 	panic(err)
			// }
			file_path := s.config.fileDir() + msg.Header["path"].(string)
			md5 := msg.Header["md5"].(string)
			file_size := msg.Header["file_size"].(int64)
			uploaded_size := msg.Header["uploaded_size"].(int64)
			server_file_id := msg.Header["server_file_id"].(string)
			block_name := uuid.New().String()
			block_path := s.config.blockDir() + block_name
			f, err := os.Create(block_path)
			if err != nil {
				panic(err)
			}
			written, err := io.CopyN(f, session, msg.BodySize)
			if err != nil {
				log.Println("receive file block failed:", err)
			} else {
				log.Println("received a new file block")
			}
			start := uploaded_size
			end := written + start
			//record uploaded file block
			parameters := url.Values{}
			parameters.Add("server_file_id", server_file_id)
			parameters.Add("name", block_name)
			parameters.Add("start", strconv.FormatInt(start, 10))
			parameters.Add("end", strconv.FormatInt(end, 10))
			rar := common.NewRestApiRequest("POST", internal.GetApiUrlPath("file-block"), []byte(parameters.Encode()))
			_, err = internal.RestApiClient().Do(rar)
			if err != nil {
				panic(err)
			}
			//assemble files if bytes of whole file has uploaded
			if end == file_size {
				//query all file blocks
				rar := common.NewRestApiRequest("GET", internal.GetApiUrlPath("file-block")+"?server_file_id="+server_file_id, nil)
				resp, err := internal.RestApiClient().Do(rar)
				if err != nil {
					panic(err)
				}
				m, err := common.ReadAsMap(resp.Body)
				if err != nil {
					panic(err)
				}
				data := m["data"].([]interface{})
				//create temp file
				temp_file_path := s.config.tempDir() + uuid.New().String()
				temp_file, err := os.Create(temp_file_path)
				if err != nil {
					panic(err)
				}
				for i := len(data) - 1; i >= 0; i-- {
					block := data[i].(map[string]interface{})
					log.Println(block)
					block_path := block["Path"].(string)
					block_file, err := os.Open(s.config.blockDir() + block_path)
					if err != nil {
						panic(err)
					}
					//copy block content to temp file
					_, err = io.Copy(temp_file, block_file)
					if err != nil {
						panic(err)
					}
					block_file.Close()
				}
				//close file stream
				temp_file.Close()
				//check md5
				tmp_file_md5, err := x.Hash_file_md5(temp_file_path)
				if err != nil {
					panic(err)
				}
				if tmp_file_md5 != md5 {
					panic("the md5 is inconsistent")
				}
				//already assembled successfully, then change temp file name to final file name
				err = os.Rename(temp_file_path, file_path)
				if err != nil {
					panic(err)
				}
				//change status
				rar = common.NewRestApiRequest("PUT", internal.GetApiUrlPath("file"), []byte(fmt.Sprintf("server_file_id=%s", server_file_id)))
				_, err = internal.RestApiClient().Do(rar)
				if err != nil {
					panic(err)
				}
			}
			reply := message.NewMessage(message.MT_Reply)
			session.Send(reply, nil)
		case message.MT_Request_Repeat:
		case message.MT_Download_File:
			file_path := s.config.fileDir() + msg.Header["path"].(string)
			err := session.SendFile(file_path, func(filename string, md5 string, size int64) (int64, bool) {
				return 0, true
			})
			if err != nil {
				panic(err)
			}
		case message.MT_DISCONNECT:
			panic(errors.New("peer requested to disconnect connections"))
		case message.MT_SYNC:

		}
	}
}
