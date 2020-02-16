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
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/swishcloud/filesync/internal"

	"golang.org/x/oauth2"
	"gopkg.in/yaml.v2"

	"github.com/google/uuid"
	"github.com/swishcloud/filesync/message"
	"github.com/swishcloud/filesync/message/models"
	"github.com/swishcloud/filesync/session"
	"github.com/swishcloud/filesync/x"
	"github.com/swishcloud/gostudy/common"
)

type FileSyncServer struct {
	config          *config
	Storage         *Storage
	skip_tls_verify bool
	httpClient      *http.Client
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
	s.Storage = NewStorage(s.config.FileLocation, "")
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
	go s.StartRepeat()
	for {
		// Wait for a connection.
		conn, err := l.Accept()
		if err != nil {
			log.Fatal(err)
		}
		// Handle the connection in a new goroutine.
		// The loop then returns to accepting, so that
		// multiple connections may be served concurrently.
		go s.serveSession(conn)
	}
}

func (s *FileSyncServer) StartRepeat() {
	for {
		s.startRepeat()
		time.Sleep(time.Second * 60 * 3)
	}
}
func (server *FileSyncServer) startRepeat() {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("repeating failed,cause:%s", err)
		}
	}()
	// if server.Repeat != "" {
	// 	log.Println("start repeating data from server addr:", server.Repeat)
	// 	conn, err := net.Dial("tcp", server.Repeat)
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	log.Println("connected successfully", server.Repeat)
	// 	s := session.NewSession(conn)
	// 	msg := new(message.Message)
	// 	msg.MsgType = message.MT_Get_All_Files
	// 	reply, err := s.Fetch(msg, nil)
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	files := []models.File{}
	// 	s.ReadJson(int(reply.BodySize), &files)
	// 	log.Println("got file list:", files)
	// 	for i := 0; i < len(files); i++ {
	// 		fileName := files[i].Path
	// 		file_path := path.Join(server.Storage.root, fileName)
	// 		if x.PathExist(file_path) {
	// 			log.Printf("%s exists,skip", file_path)
	// 			continue
	// 		}
	// 		if files[i].IsFolder {
	// 			err = os.Mkdir(file_path, os.ModePerm)
	// 			if err != nil {
	// 				panic(err)
	// 			}
	// 			log.Println("create folder:", file_path)
	// 			checkFile(files[i], file_path)
	// 			continue
	// 		}
	// 		msg.MsgType = message.MT_Download_File
	// 		err := s.Send(msg, fileName)
	// 		if err != nil {
	// 			panic(err)
	// 		}
	// 		reply, err := s.ReadMessage()
	// 		if err != nil {
	// 			panic(err)
	// 		}
	// 		log.Println("downloading file:", fileName)
	// 		_, err = s.ReadFile(file_path, reply.Header["md5"].(string), reply.BodySize)
	// 		if err != nil {
	// 			panic(err)
	// 		}
	// 		checkFile(files[i], file_path)
	// 		log.Println("received file:", fileName)
	// 	}
	// 	s.Close()
	// 	log.Println("finished repeating")
	// }
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
func (s *FileSyncServer) serveSession(c net.Conn) {
	session := session.NewSession(c)
	defer func() {
		if err := recover(); err != nil {
			log.Println("close session:", session, "cause:", err)
			session.Close()
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
			_, token, err := s.checkToken(msg)
			if err != nil {
				panic(err)
			}
			file_name := msg.Header["file_name"].(string)
			md5 := msg.Header["md5"].(string)
			directory_path := msg.Header["directory_path"].(string)
			is_hidden, err := strconv.ParseBool(msg.Header["is_hidden"].(string))
			if err != nil {
				panic(err)
			}
			data, err := internal.GetFileData(file_name, md5, directory_path, is_hidden, token)
			if err != nil {
				panic(err)
			}
			file_path := s.config.fileDir() + data["Path"].(string)
			server_file_id := data["Server_file_id"].(string)
			uploaded_size := int64(data["Uploaded_size"].(float64))
			file_size := int64(data["Size"].(float64))
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
			file_path := msg.Header["path"].(string)
			err = session.SendFile(path.Join(s.config.fileDir(), file_path), nil)
			if err != nil {
				panic(err)
			}
			log.Println("sent file:", file_path)
		case message.MT_Get_All_Files:
			files, err := s.Storage.GetFiles(s.Storage.root, "")
			if err != nil {
				panic(err)
			}
			reply_msg := new(message.Message)
			reply_msg.MsgType = message.MT_Reply

			err = session.Send(reply_msg, files)
			if err != nil {
				panic(err)
			}
		case message.MT_DISCONNECT:
			panic(errors.New("peer requested to disconnect connections"))
		}

	}
}

type Storage struct {
	root    string
	filters []string
}

//s.Ack()
func NewStorage(root, filters string) *Storage {
	storage := new(Storage)
	storage.root = root
	if filters != "" {
		storage.filters = strings.Split(filters, ";")
	} else {
		storage.filters = []string{}
	}
	return storage
}
func (s *Storage) Ignore(path string) bool {
	for i := 0; i < len(s.filters); i++ {
		if filepath.Base(path) == s.filters[i] {
			return true
		}
	}
	return false
}
func (s *Storage) GetFiles(p string, prefix string) ([]models.File, error) {
	fileInfos, err := ioutil.ReadDir(p)
	if err != nil {
		return nil, err
	}
	files := []models.File{}
	for i := 0; i < len(fileInfos); i++ {
		file := models.File{IsFolder: fileInfos[i].IsDir(), Path: prefix + fileInfos[i].Name()}
		fullPath := s.root + "/" + file.Path
		if s.Ignore(fullPath) {
			log.Printf("Ignore file or directory:%s", fullPath)
			continue
		}
		file.IsHidden, err = x.IsHidden(fullPath)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
		if file.IsFolder {
			fs, err := s.GetFiles(fullPath, file.Path+"/")
			if err != nil {
				return nil, err
			}
			files = append(files, fs...)
		}
	}
	return files, nil
}
