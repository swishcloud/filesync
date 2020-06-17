package models

import "time"

type Log struct {
	Insert_time time.Time
	P_id        *string
	Action      int
	Number      int64
	File_id     string
	File_type   int
	File_name   string
	File_md5    *string
	File_size   *int64
}
type File struct {
	Id           string
	Name         string
	File_type    int
	Status       int
	Md5          *string
	Download_md5 *string
	Server_md5   *string
	Is_merged    bool
	Is_deleted   *bool
}
