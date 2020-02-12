package client

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"strconv"

	"github.com/swishcloud/filesync/internal"

	"github.com/swishcloud/filesync/auth"
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
func getFileData(file_name, md5 string, skip_tls_verify bool) map[string]interface{} {
	params := url.Values{}
	params.Add("md5", md5)
	params.Add("name", file_name)
	url := x.GetApiUrlPath("file") + "?" + params.Encode()
	rac := common.NewRestApiClient("GET", url, nil, skip_tls_verify)
	resp, err := rac.Do()
	if err != nil {
		panic(err)
	}
	m := common.ReadAsMap(resp.Body)
	if m["error"] != nil {
		panic(m["error"].(string))
	}
	if m["data"] == nil {
		return nil
	}
	return m["data"].(map[string]interface{})
}
func SendFile(file_path string, skip_tls_verify bool) error {
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
	msg.Header[internal.TokenHeaderKey] = auth.GetToken().AccessToken

	data := getFileData(file_info.Name(), md5, skip_tls_verify)
	is_completed := false
	reused := false
	if data == nil {
		//need to insert file record
		rac := common.NewRestApiClient("POST", x.GetApiUrlPath("file"), []byte("name="+name+"&md5="+md5+"&size="+strconv.FormatInt(size, 10)), false).SetAuthHeader(auth.GetToken())
		resp, err := rac.Do()
		if err != nil {
			panic(err)
		}
		m := common.ReadAsMap(resp.Body)
		if m["error"] != nil {
			return errors.New(m["error"].(string))
		}
		data = getFileData(name, md5, skip_tls_verify)
		if data == nil {
			return errors.New("still no file record")
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

	conn, err := net.Dial("tcp", ip+":"+strconv.FormatInt(port, 10))
	if err != nil {
		return err
	}
	s := session.NewSession(conn)
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

func DownloadFile(file_id string, skip_tls_verify bool) error {
	rac := common.NewRestApiClient("GET", x.GetApiUrlPath("file")+"?file_id="+file_id, nil, skip_tls_verify)
	resp, err := rac.Do()
	if err != nil {
		panic(err)
	}
	m := common.ReadAsMap(resp.Body)
	if m["error"] != nil {
		panic(m["error"].(string))
	}
	data := m["data"].(map[string]interface{})
	Ip := data["Ip"].(string)
	Port := data["Port"].(float64)
	Path := data["Path"].(string)
	file_name := data["Name"].(string)
	address := fmt.Sprintf("%s:%.0f", Ip, Port)

	conn, err := net.Dial("tcp", address)
	if err != nil {
		log.Fatal(err)
	}
	s := session.NewSession(conn)
	msg := message.NewMessage(message.MT_Download_File)
	msg.Header["path"] = Path
	file_msg, err := s.Fetch(msg, nil)
	if err != nil {
		return err
	}
	filepath := file_name
	_, err = s.ReadFile(filepath, file_msg.Header["md5"].(string), file_msg.BodySize)
	if err != nil {
		panic(err)
	}
	log.Println("downloaded file:", file_name)
	return nil
}
