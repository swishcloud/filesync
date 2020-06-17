package client

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/swishcloud/filesync/internal"
	"github.com/swishcloud/filesync/message"
	"github.com/swishcloud/filesync/x"
	"github.com/swishcloud/gostudy/common"

	"github.com/swishcloud/filesync/session"
)

func Ping(server string) {
	conn, err := net.Dial("tcp", server)
	if err != nil {
		log.Fatal(err)
	}
	s := session.NewSession(conn)
	msg := new(message.Message)
	msg.MsgType = message.MT_PING
	for i := 0; i < 5; i++ {
		err := s.Send(msg, nil)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(msg.MsgType)
		m, err := s.ReadMessage()
		if err != nil {
			log.Fatal(err)
		}
		if m.MsgType != message.MT_PANG {
			log.Fatal("unexpected response type:", m.MsgType)
		}
		fmt.Println(m.MsgType)
	}
	fmt.Println("server status is ok")
}
func SendFile(file_path, directory_path string, is_hidden bool) error {
	msg := message.NewMessage(message.MT_FILE)
	md5, err := x.Hash_file_md5(file_path)
	if err != nil {
		return err
	}
	f, err := os.Open(file_path)
	defer f.Close()
	if err != nil {
		return err
	}
	file_info, err := f.Stat()
	if err != nil {
		return err
	}
	name := file_info.Name()
	size := file_info.Size()
	msg.Header["md5"] = md5
	msg.Header["file_name"] = name
	msg.Header["directory_path"] = directory_path
	msg.Header["is_hidden"] = strconv.FormatBool(is_hidden)
	token, err := internal.GetToken()
	if err != nil {
		return err
	}
	msg.Header[internal.TokenHeaderKey] = token.AccessToken

	data, err := internal.GetFileData(name, md5, directory_path, is_hidden, token)
	if err != nil {
		return err
	}
	is_completed := false
	reused := false
	need_create_file := false
	if data == nil {
		need_create_file = true
	} else {
		if strings.Trim(data["Md5"].(string), " ") != md5 {
			need_create_file = true
			//default to delete the existing file
			if err := internal.DeleteFile(data["File_id"].(string)); err != nil {
				return err
			}
		}
	}
	if need_create_file {
		//need to insert file record
		insert_parameters := url.Values{}
		insert_parameters.Add("name", name)
		insert_parameters.Add("md5", md5)
		insert_parameters.Add("size", strconv.FormatInt(size, 10))
		insert_parameters.Add("directory_path", directory_path)
		insert_parameters.Add("is_hidden", strconv.FormatBool(is_hidden))
		rar := common.NewRestApiRequest("POST", x.GetApiUrlPath("file"), []byte(insert_parameters.Encode())).SetAuthHeader(token)
		resp, err := internal.RestApiClient().Do(rar)
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
		data, err = internal.GetFileData(name, md5, directory_path, is_hidden, token)
		if err != nil {
			return err
		}
		is_completed = data["Is_completed"].(bool)
		if is_completed {
			reused = true
		}
	}
	if reused {
		fmt.Println("successfully uploaded")
		return nil
	}
	is_completed = data["Is_completed"].(bool)
	ip := data["Ip"].(string)
	port := int64(data["Port"].(float64))
	if is_completed {
		fmt.Println("this file already exists")
		return nil
	}
	uploaded_size := int64(data["Uploaded_size"].(float64))
	fmt.Println("ready to upload")
	s, err := sessionFactory(ip + ":" + strconv.FormatInt(port, 10))
	if err != nil {
		return err
	}
	_, err = f.Seek(uploaded_size, 1)
	if err != nil {
		return err
	}
	err = s.SendMessage(msg, f, size-uploaded_size)
	if err != nil {
		return err
	}
	_, err = s.ReadMessage()
	if err != nil {
		return err
	}
	fmt.Println("successfully uploaded")
	return nil
}
func TransferDirectory(path string, pre_transfer func(path string) (ok bool)) error {
	path, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	path = strings.ReplaceAll(path, "\\", "/")
	path = strings.TrimSuffix(path, "/")
	p_path := regexp.MustCompile(".*/").FindString(path)
	items := []*common.FileInfoWrapper{}
	err = common.ReadAllFiles(path, &items)
	if err != nil {
		return err
	}
	failureNum := 0
	skipNum := 0
	for index, item := range items {
		location := strings.Replace(item.Path, p_path, "", 1)
		location = regexp.MustCompile(".*/").FindString(location)
		location = strings.TrimSuffix(location, "/")
		fmt.Println("target location:", location)
		is_hidden, err := x.IsHidden(item.Path)
		if err != nil {
			log.Printf(err.Error())
			failureNum++
		} else {
			if !pre_transfer(item.Path) {
				skipNum++
				fmt.Printf("skiped %s\r\n", item.Path)
				continue
			}
			if item.Fi.IsDir() {
				fmt.Printf("found folder '%s'\r\n", item.Path)
				//ensure directory already created
				if err := internal.CreateDirectory(location, item.Fi.Name(), is_hidden); err != nil {
					failureNum++
					return err
				}
			} else {
				fmt.Printf("uploading file '%s'\r\n", item.Path)
				err = SendFile(item.Path, location, is_hidden)
				if err != nil {
					log.Printf(err.Error())
					failureNum++
				}
			}
		}
		fmt.Printf("progress: %d/%d failure:%d skip:%d\r\n", index+1, len(items), failureNum, skipNum)
	}
	return nil
}

var sessions map[string]*session.Session = map[string]*session.Session{}

func sessionFactory(addr string) (*session.Session, error) {
	if sessions[addr] == nil {
		log.Println("create new tcp connection to " + addr)
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			return nil, err
		}
		s := session.NewSession(conn)
		sessions[addr] = s
	}
	return sessions[addr], nil
}

func DownloadFile(file_id string, save_path string) error {
	token, err := internal.GetToken()
	if err != nil {
		return err
	}
	rar := common.NewRestApiRequest("GET", internal.GetApiUrlPath("file")+"?file_id="+file_id, nil).SetAuthHeader(token)
	resp, err := internal.RestApiClient().Do(rar)
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
	data := m["data"].(map[string]interface{})
	Ip := data["Ip"].(string)
	Port := data["Port"].(float64)
	Path := data["Path"].(string)
	file_name := data["Name"].(string)
	address := fmt.Sprintf("%s:%.0f", Ip, Port)

	conn, err := net.Dial("tcp", address)
	if err != nil {
		return err
	}
	s := session.NewSession(conn)
	msg := message.NewMessage(message.MT_Download_File)
	msg.Header["path"] = Path
	msg.Header[internal.TokenHeaderKey] = token.AccessToken
	file_msg, err := s.Fetch(msg, nil)
	if err != nil {
		return err
	}
	if save_path == "" {
		save_path = file_name
	}
	_, err = s.ReadFile(save_path, file_msg.Header["md5"].(string), file_msg.BodySize)
	if err != nil {
		return err
	}
	log.Println("downloaded file:", save_path)
	return nil
}
