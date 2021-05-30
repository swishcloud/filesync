package cmd

import (
	"log"
	"net"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/swishcloud/filesync/client"
	"github.com/swishcloud/filesync/message"

	"github.com/spf13/cobra"
	"github.com/swishcloud/filesync/internal"
	"github.com/swishcloud/filesync/session"
	"github.com/swishcloud/filesync/storage"
	"github.com/swishcloud/gostudy/common"
)

var local_changed = false
var db_file_path = ""
var local_max = 0
var syncCmd = &cobra.Command{
	Use: "sync",
	Run: func(cmd *cobra.Command, args []string) {
		path, err := cmd.Flags().GetString("path")
		if err != nil {
			log.Fatal(err)
		}
		db_file_path = filepath.Join(filepath.Dir(path), "filesync.db")
		if false {
			_, err := os.Stat(db_file_path)
			if !os.IsNotExist(err) {
				err = os.Remove(db_file_path)
				if err != nil {
					panic(err)
				}
			}
			println("deleted db file")
		}

		go check_file_change(path)
		receive(path, db_file_path)
	},
}

func check_file_change(rootpath string) {
	for {
		local_changed = true
		time.Sleep(time.Second * 5)
	}
}
func receive(rootpath string, db_file_path string) {
	for {
		log.Println("create new tcp connection to " + internal.GlobalConfig().WebServerTcpAddess)
		conn, err := net.Dial("tcp", internal.GlobalConfig().WebServerTcpAddess)
		if err != nil {
			prepareReconnect()
			continue
		}
		s := session.NewSession(conn)
		s.Send(message.NewMessage(message.MT_SYNC), nil)
		for {
			msg, err := s.ReadMessage()
			if err != nil {
				prepareReconnect()
				break
			}
			storage := storage.NewSQLManager(db_file_path)
			local_max := storage.MaxNumber()
			if err := storage.Commit(); err != nil {
				panic(err)
			}
			server_max := msg.Header["max"].(int64)
			if err != nil {
				panic(err)
			}
			if server_max > local_max {
				fetch(db_file_path, local_max+1)
				pre_sync(rootpath)
				sync(rootpath)
				if err := check_upload_local(rootpath); err != nil {
					log.Println(err)
				}
			} else if local_changed {
				local_changed = false
				sync(rootpath)
				if err := check_upload_local(rootpath); err != nil {
					log.Println(err)
				}
			}
		}
	}
}
func prepareReconnect() {
	var n int64 = 5
	log.Printf("connection disconnected,reconnect in %d s", n)
	time.Sleep(time.Second * time.Duration(n))
}
func fetch(db_file_path string, start int64) {
	storage := storage.NewSQLManager(db_file_path)
	defer func() {
		err := storage.Commit()
		if err != nil {
			panic(err)
		}
	}()
	log.Printf("start fetching server files...")
	logs, err := internal.GetLogs(start)
	if err != nil {
		log.Fatal(err)
	}
	for _, log := range logs {
		if log.File_type == 1 {
			//file
			switch log.Action {
			case 1:
				storage.AddFile(log, time.Now().UTC())
			case 2:
				//delete
				storage.Delete(log.File_id)
			case 3:
				//update
				storage.UpdateFile(log)
			case 4:
				//rename
			default:
				panic("unknow action")
			}
		} else if log.File_type == 2 {
			//directory
			switch log.Action {
			case 1:
				//add
				storage.AddFile(log, time.Now().UTC())
			case 2:
				//delete
				storage.Delete(log.File_id)
			case 3:
				//update
				storage.UpdateFile(log)
			case 4:
				//rename
			default:
				panic("unknow action")
			}
		}
		storage.SetMerge(log.File_id, false)
		storage.UpdateLastNumber(log.Number)
	}
	log.Printf("finished fetching server files...")
}
func pre_sync(rootpath string) {
	store := storage.NewSQLManager(db_file_path)
	files := store.GetFiles()
	store.Commit()
	for _, file := range files {
		store = storage.NewSQLManager(db_file_path)
		filepath := store.GetFilePath(file.Id)
		filepath = path.Join(rootpath, filepath)
		if file.Is_deleted != nil && *file.Is_deleted { //the server file has been deleted,if local file exists should delete it accordingly.
			p := string([]rune(filepath)[len(rootpath):])
			if file := store.GetFileByPath(p); file == nil {
				if err := os.Remove(filepath); err != nil && !os.IsNotExist(err) {
					log.Printf("delete directory %s failed", filepath)
				}
			}
			store.SetMerge(file.Id, true)
		} else if file.File_type == 1 { //file
			if exists, err := common.CheckIfFileExists(filepath); err != nil {
				panic(err)
			} else {
				if exists { //the local file exists.
					if hash, err := common.FileMd5Hash(filepath); err != nil {
						panic(err)
					} else {
						if hash == strings.TrimRight(*file.Server_md5, " ") { //the local file has not been modified.
							store.SetMerge(file.Id, true)
						} else { //the local file has been modified
							if file.Download_md5 == nil || *file.Download_md5 != *file.Server_md5 {
								//the file is not modified based on the latest reversion,should rename it as another copy of it
								rename()
								store.SetAsAvailableOnline(file.Id)
							} else {
								store.UpdateLocalMd5(file.Id, hash)
							}
							store.SetMerge(file.Id, true)
						}
					}
				} else { //the local file does not exists, merge it directly.
					store.SetMerge(file.Id, true)
				}
			}
		} else if file.File_type == 2 { //directory
			store.SetMerge(file.Id, true)
		}
		store.Commit()
	}
}
func sync(rootpath string) {
	store := storage.NewSQLManager(db_file_path)
	files := store.GetMergedFiles()
	store.Commit()
	for _, file := range files {
		store = storage.NewSQLManager(db_file_path)
		if file.Is_deleted != nil && *file.Is_deleted { //the server file has been deleted,deleting the local file has done in pre_sync,just delete db record here.
			store.HardDelete(file.Id)
			store.Commit()
			continue
		}
		relative_filepath := store.GetFilePath(file.Id)
		location := regexp.MustCompile(".*/").FindString(relative_filepath)
		location = strings.TrimSuffix(location, "/")
		filepath := path.Join(rootpath, relative_filepath)
		if exist, err := common.CheckIfFileExists(filepath); err != nil {
			log.Println(err)
		} else {
			if file.File_type == 1 { //file
				if file.Download_md5 == nil || !compareMd5(*file.Download_md5, *file.Server_md5) { //the file has not been download,or has updates,just download it to local
					if err := os.MkdirAll(path.Dir(filepath), os.ModePerm); err != nil {
						panic(err)
					}
					need_download := true
					if exist {
						if hash, err := common.FileMd5Hash(filepath); err != nil {
							log.Println(err)
							store.Commit()
							continue
						} else {
							if compareMd5(hash, *file.Server_md5) {
								need_download = false
							}
						}
					}
					if need_download {
						if err := client.DownloadFile(file.Id, filepath); err != nil {
							log.Println(err)
							store.Commit()
							continue
						}
					}
					fi, err := os.Stat(filepath)
					if err != nil {
						panic(err)
					}
					store.UpdateDownloadMd5(file.Id, *file.Server_md5)
					store.UpdateLocalMd5(file.Id, *file.Server_md5)
					store.UpdateModifyTime(file.Id, fi.ModTime().UTC())
				} else if file.Md5 != nil && !compareMd5(*file.Md5, *file.Download_md5) { //the file has changes on local,upload it server
					if err := client.SendFile(filepath, location, false); err != nil {
						log.Println(err)
					}

				} else if file.Download_md5 != nil {
					if !exist {
						if err := internal.DeleteFile(file.Id); err != nil {
							log.Println(err)
						}
					}
				}
			} else { //directory
				if err := os.MkdirAll(filepath, os.ModePerm); err != nil {
					panic(err)
				}
			}
		}
		store.Commit()
	}
}
func check_upload_local(root_path string) error {
	store := storage.NewSQLManager(db_file_path)
	defer store.Commit()
	items := []*common.FileInfoWrapper{}
	if err := common.ReadAllFiles(root_path, &items); err != nil {
		return err
	}
	items = items[1:]
	for _, item := range items {
		runes := []rune(item.Path)
		p := string(runes[len(root_path):])
		location := regexp.MustCompile(".*/").FindString(p)
		location = strings.Trim(location, "/")
		file := store.GetFileByPath(p)
		if file == nil {
			if item.Fi.IsDir() {
				if err := internal.CreateDirectory(location, item.Fi.Name(), false); err != nil {
					log.Fatal(err)
				}
			} else if err := client.SendFile(item.Path, location, false); err != nil {
				return err
			}
		} else {
			if item.Fi.IsDir() {

			} else {
				if hash, err := common.FileMd5Hash(item.Path); err != nil {
					log.Println(err)
				} else {
					if !compareMd5(hash, *file.Server_md5) {
						if err := client.SendFile(item.Path, location, false); err != nil {
							return err
						}
					}
				}
			}
		}
	}
	return nil
}
func compareMd5(left, right string) bool {
	return strings.TrimSpace(left) == strings.TrimSpace(right)
}
func rename() {

}
func init() {
	rootCmd.AddCommand(syncCmd)
	syncCmd.Flags().String("path", "", "the path of location to sync file")
	syncCmd.MarkFlagRequired("path")
}
