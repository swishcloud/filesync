package server

import (
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path"
	"time"

	"github.com/swishcloud/filesync/message"
	"github.com/swishcloud/filesync/session"
	"github.com/swishcloud/filesync/x"
)

type FileSyncServer struct {
	Port    string
	Repeat  string
	Storage *Storage
}

func NewFileSyncServer(port string, root string, repeat string) *FileSyncServer {
	s := &FileSyncServer{Port: port, Repeat: repeat}
	s.Storage = NewStorage(root)
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
		files := []string{}
		s.ReadJson(int(reply.BodySize), &files)
		log.Println("got file list:", files)
		for i := 0; i < len(files); i++ {
			fileName := files[i]
			file_path := path.Join(server.Storage.root, fileName)
			if x.PathExist(file_path) {
				log.Printf("%s exists,skip", file_path)
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
			log.Println("received file:", fileName)
		}
		s.Close()
		log.Println("finished repeating")
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
			fileInfos, err := s.Storage.GetFiles()
			if err != nil {
				panic(err)
			}
			files := []string{}
			for i := 0; i < len(fileInfos); i++ {
				if !fileInfos[i].IsDir() {
					files = append(files, fileInfos[i].Name())
				}
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
	root string
}

//s.Ack()
func NewStorage(root string) *Storage {
	storage := new(Storage)
	storage.root = root
	return storage
}

func (s *Storage) GetFiles() ([]os.FileInfo, error) {
	files, err := ioutil.ReadDir(s.root)
	if err != nil {
		return nil, err
	}
	return files, nil
}
