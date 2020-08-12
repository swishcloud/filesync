package cmd

import (
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/swishcloud/filesync/internal"
	"github.com/swishcloud/filesync/message"
	"github.com/swishcloud/filesync/session"
	"github.com/swishcloud/filesync/storage/models"

	"github.com/swishcloud/gostudy/common"

	"github.com/spf13/cobra"
)

// var uploadCmd = &cobra.Command{
// 	Use: "upload",
// 	Run: func(cmd *cobra.Command, args []string) {
// 		file_path, err := cmd.Flags().GetString("file_path")
// 		file_path, err = filepath.Abs(file_path)
// 		if err != nil {
// 			log.Fatal(err)
// 		}
// 		if err != nil {
// 			log.Fatal(err)
// 		}
// 		file_path = strings.ReplaceAll(file_path, "\\", "/")
// 		file_path = strings.TrimSuffix(file_path, "/")
// 		p_file_path := regexp.MustCompile(".*/").FindString(file_path)
// 		items := []*common.FileInfoWrapper{}
// 		err = common.ReadAllFiles(file_path, &items)
// 		if err != nil {
// 			log.Fatal(err)
// 		}
// 		failureNum := 0
// 		for index, item := range items {
// 			location := strings.Replace(item.Path, p_file_path, "", 1)
// 			location = regexp.MustCompile(".*/").FindString(location)
// 			location = strings.TrimSuffix(location, "/")
// 			fmt.Println("target location:", location)
// 			is_hidden, err := x.IsHidden(item.Path)
// 			if err != nil {
// 				log.Printf(err.Error())
// 				failureNum++
// 			} else {
// 				if item.Fi.IsDir() {
// 					fmt.Printf("found folder '%s'\r\n", item.Path)
// 					//ensure directory already created
// 					if err := internal.CreateDirectory(location, item.Fi.Name(), is_hidden); err != nil {
// 						log.Fatal(err.Error())
// 						failureNum++
// 					}
// 				} else {
// 					fmt.Printf("uploading file '%s'\r\n", item.Path)
// 					err = client.SendFile(item.Path, location, is_hidden)
// 					if err != nil {
// 						log.Printf(err.Error())
// 						failureNum++
// 					}
// 				}
// 			}
// 			fmt.Printf("progress: %d/%d failure:%d\r\n", index+1, len(items), failureNum)
// 		}
// 	},
// }
var uploadCmd = &cobra.Command{
	Use: "upload",
	Run: func(cmd *cobra.Command, args []string) {
		file_path, err := cmd.Flags().GetString("file_path")
		internal.CheckErr(err)
		location := "/"
		file_path, err = filepath.Abs(file_path)
		internal.CheckErr(err)
		root_path := filepath.Dir(file_path)
		root_path = strings.TrimRight(root_path, "\\") //the windows root directory will remains trailing slashes,for example:C:\
		fmt.Println(`will upload ` + file_path)
		files := []*common.FileInfoWrapper{}
		err = common.ReadAllFiles(file_path, &files)
		internal.CheckErr(err)
		filePaths := []string{}
		folderPaths := []string{}
		for _, file := range files {
			if file.Fi.IsDir() {
				folderPaths = append(folderPaths, file.Path)
			} else {
				filePaths = append(filePaths, file.Path)
			}
		}
		file_actions, err := uploadFiles(filePaths, root_path, location)
		internal.CheckErr(err)
		folder_actions := []models.FileAction{}
		for _, folder_path := range folderPaths {
			p := getServerPath(folder_path, root_path, location)
			fa := models.FileAction{}
			fa.ActionType = 1
			fa.Md5 = ""
			fa.FileType = 2
			fa.Path = p
			folder_actions = append(folder_actions, fa)
		}
		actions := append(folder_actions, file_actions...)
		err = internal.HttpPostFileAction(actions)
		internal.CheckErr(err)
	},
}

func getServerPath(file_path, local_root_path, server_location string) string {
	p := filepath.Join(server_location, file_path[len(local_root_path)+1:])
	p = strings.ReplaceAll(p, "\\", "/")
	return p
}
func uploadFiles(files []string, local_root_path, server_location string) (actions []models.FileAction, err error) {
	for _, file_path := range files {
		md5, err := common.FileMd5Hash(file_path)
		if err != nil {
			return nil, err
		}

		fa := models.FileAction{}
		fa.ActionType = 1
		fa.FileType = 1
		fa.Md5 = md5
		fa.Path = getServerPath(file_path, local_root_path, server_location)
		actions = append(actions, fa)

		fmt.Println(`file:`, file_path)
		f, err := os.Open(file_path)
		if err != nil {
			return nil, err
		}
		fi, err := f.Stat()
		if err != nil {
			return nil, err
		}
		size := fi.Size()
		file_info, err := internal.FileInfo(md5, size)
		if err != nil {
			return nil, err
		}
		ip := file_info["ip"].(string)
		port, err := strconv.Atoi(file_info["port"].(string))
		if err != nil {
			return nil, err
		}
		is_completed, err := strconv.ParseBool(file_info["is_completed"].(string))
		if err != nil {
			return nil, err
		}
		if is_completed {
			fmt.Println("this file already exists")
			continue
		}
		uploaded_size, err := strconv.ParseInt(file_info["uploaded_size"].(string), 10, 64)
		if err != nil {
			return nil, err
		}
		fmt.Println("ready to upload")
		defer f.Close()
		_, err = f.Seek(uploaded_size, 1)
		if err != nil {
			return nil, err
		}
		msg := message.NewMessage(message.MT_FILE)
		msg.Header["path"] = file_info["path"].(string)
		msg.Header["md5"] = md5
		msg.Header["file_size"] = size
		msg.Header["uploaded_size"] = uploaded_size
		msg.Header["server_file_id"] = file_info["server_file_id"].(string)
		token, err := internal.GetToken()
		if err != nil {
			return nil, err
		}
		msg.Header[internal.TokenHeaderKey] = token.AccessToken
		s, err := sessionFactory(ip + ":" + strconv.Itoa(port))
		if err != nil {
			return nil, err
		}
		err = s.SendMessage(msg, f, size-uploaded_size)
		if err != nil {
			return nil, err
		}
		_, err = s.ReadMessage()
		if err != nil {
			return nil, err
		}
		fmt.Println("successfully uploaded")
	}
	return actions, nil
}

func init() {
	rootCmd.AddCommand(uploadCmd)
	uploadCmd.Flags().String("file_path", "", "the path of file to upload")
	uploadCmd.MarkFlagRequired("file_path")
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
