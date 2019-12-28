package server

import (
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/swishcloud/filesync/message"
	"github.com/swishcloud/filesync/message/models"
	"github.com/swishcloud/filesync/session"
	"github.com/swishcloud/filesync/x"
)

type FileSyncServer struct {
	Port    string
	Filters string
	Repeat  string
	Storage *Storage
}

func NewFileSyncServer(port string, root string, repeat string, filters string) *FileSyncServer {
	s := &FileSyncServer{Port: port, Repeat: repeat, Filters: filters}
	s.Storage = NewStorage(root, filters)
	return s
}
func (s *FileSyncServer) Serve() {
	// Listen on TCP port 2000 on all available unicast and
	// anycast IP addresses of the local system.
	l, err := net.Listen("tcp", ":"+s.Port)
	log.Println("accepting tcp connections on port", s.Port)
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
	if server.Repeat != "" {
		log.Println("start repeating data from server addr:", server.Repeat)
		conn, err := net.Dial("tcp", server.Repeat)
		if err != nil {
			panic(err)
		}
		log.Println("connected successfully", server.Repeat)
		s := session.NewSession(conn)
		msg := new(message.Message)
		msg.MsgType = message.MT_Get_All_Files
		reply, err := s.Fetch(msg, nil)
		if err != nil {
			panic(err)
		}
		files := []models.File{}
		s.ReadJson(int(reply.BodySize), &files)
		log.Println("got file list:", files)
		for i := 0; i < len(files); i++ {
			fileName := files[i].Path
			file_path := path.Join(server.Storage.root, fileName)
			if x.PathExist(file_path) {
				log.Printf("%s exists,skip", file_path)
				continue
			}
			if files[i].IsFolder {
				err = os.Mkdir(file_path, os.ModePerm)
				if err != nil {
					panic(err)
				}
				log.Println("create folder:", file_path)
				checkFile(files[i], file_path)
				continue
			}
			msg.MsgType = message.MT_Download_File
			err := s.Send(msg, fileName)
			if err != nil {
				panic(err)
			}
			reply, err := s.ReadMessage()
			if err != nil {
				panic(err)
			}
			log.Println("downloading file:", fileName)
			_, err = s.ReadFile(file_path, reply.Header["md5"].(string), reply.BodySize)
			if err != nil {
				panic(err)
			}
			checkFile(files[i], file_path)
			log.Println("received file:", fileName)
		}
		s.Close()
		log.Println("finished repeating")
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
			f, err := os.Create("temp_fle")
			if err != nil {
				panic(err)
			}
			_, err = io.Copy(f, c)
			if err != nil {
				log.Println("receive file failed")
				panic(err)
			} else {
				log.Println("received a new file")
			}
		case message.MT_Request_Repeat:
		case message.MT_Download_File:
			log.Println("getting file name")
			file := ""
			err := session.ReadJson(int(msg.BodySize), &file)
			if err != nil {
				panic(err)
			}
			err = session.SendFile(path.Join(s.Storage.root, file))
			if err != nil {
				panic(err)
			}
			log.Println("sent file:", file)
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
