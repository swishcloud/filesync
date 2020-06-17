package storage

import (
	"database/sql"
	dbsqql "database/sql"
	"log"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/swishcloud/filesync/storage/models"
	"github.com/swishcloud/gostudy/tx"
)

type SQLITEManager struct {
	Tx *tx.Tx
}

var db *sql.DB

func NewSQLManager(db_conn_info string) *SQLITEManager {
	if db == nil {
		d, err := sql.Open("sqlite3", db_conn_info)
		if err != nil {
			log.Fatal(err)
		}
		db = d
		_, err = db.Exec(`create table IF NOT EXISTS file  
		(
			id TEXT,
			p_id TEXT,
			type TEXT,
			md5 TEXT,
			name TEXT,
			modify_time TEXT,
			size INTEGER,
			status TEXT,
			server_md5 TEXT,
			download_md5 TEXT,
			is_merged int,
			is_deleted int
		);
		
		`)
		if err != nil {
			log.Fatal(err)
		}
		_, err = db.Exec("CREATE TABLE IF NOT EXISTS sys (last_number INTEGER)")
		if err != nil {
			log.Fatal(err)
		}
		_, err = db.Exec("insert into sys(last_number) values(-1)")
		if err != nil {
			log.Fatal(err)
		}
	}
	tx, err := tx.NewTx(db)
	if err != nil {
		panic(err)
	}
	return &SQLITEManager{Tx: tx}
}
func (m *SQLITEManager) Commit() error {
	return m.Tx.Commit()
}
func (m *SQLITEManager) Rollback() error {
	return m.Tx.Rollback()
}
func (m *SQLITEManager) Initialize() {
	conn_str := "file:test.db"
	db, err := sql.Open("sqlite3", conn_str)
	if err != nil {
		panic(err)
	}
	m.Tx, err = tx.NewTx(db)
	if err != nil {
		panic(err)
	}
	err = db.Ping()
	if err != nil {
		panic(err)
	}

}
func (m *SQLITEManager) MaxNumber() (n int64) {
	query := `SELECT max(last_number)
	FROM sys`
	rows := m.Tx.MustQuery(query)
	defer rows.Close()
	if rows.Next() {
		var number *int64
		rows.MustScan(&number)
		if number == nil {
			return -1
		}
		return *number
	}
	return -1
}

func (m *SQLITEManager) AddFile(log models.Log, modify_time time.Time) {
	m.Tx.MustExec("insert into file(id,p_id,type,name,modify_time,size,status,server_md5,is_merged) values (?,?,?,?,?,?,?,?,0)", log.File_id, log.P_id, log.File_type, log.File_name, modify_time, log.File_size, 1, log.File_md5)
}
func (m *SQLITEManager) UpdateFile(log models.Log) {
	m.Tx.MustExec("update file set server_md5=? where id=?", log.File_md5, log.File_id)
}

func (m *SQLITEManager) SetAsAvailableOnline(id string) {
	m.Tx.MustExec("update file set download_md5=?,md5=? where id=?", nil, nil, id)
}
func (m *SQLITEManager) UpdateLocalMd5(id string, md5 string) {
	m.Tx.MustExec("update file set md5=$1 where id=$2", md5, id)
}

func (m *SQLITEManager) UpdateDownloadMd5(id string, md5 string) {
	m.Tx.MustExec("update file set download_md5=$1 where id=$2", md5, id)
}

func (m *SQLITEManager) SetMerge(id string, is_merged bool) {
	m.Tx.MustExec("update file set is_merged=$1 where id=$2", is_merged, id)
}

func (m *SQLITEManager) Delete(id string) {
	m.Tx.MustExec("update file set is_deleted=1 where id=$1", id)
}

func (m *SQLITEManager) HardDelete(id string) {
	m.Tx.MustExec("delete from file where id=$1", id)
}
func (m *SQLITEManager) SetFileStatus(log models.Log, status int) {
	m.Tx.MustExec("update file set status=? where id=?", status, log.File_id)
}
func (m *SQLITEManager) UpdateLastNumber(last_number int64) {
	m.Tx.MustExec("update sys set last_number=?", last_number)
}
func (m *SQLITEManager) UpdateModifyTime(file_id string, modify_time time.Time) {
	m.Tx.MustExec("update file set modify_time=? where id=?", modify_time, file_id)
}
func (m *SQLITEManager) DeleteFile(file_id string) {
	m.Tx.MustExec("delete from file where id=?", file_id)
}

func (m *SQLITEManager) GetFilePath(file_id string) string {
	path := ""
	p_id := &file_id
	for p_id != nil {
		query := `select name,p_id from file where id=?`
		row := m.Tx.MustQueryRow(query, p_id)
		var name string
		row.MustScan(&name, &p_id)
		if p_id != nil {
			path = name + "/" + path
		}
	}
	return strings.TrimSuffix(path, "/")
}

func (m *SQLITEManager) getFiles(where string, args ...interface{}) []models.File {
	query := `WITH RECURSIVE CTE as(select *,0 as n from file where p_id is null
		union all
		select file.*,CTE.n+1 from file inner join CTE on file.p_id=CTE.id
		)
	select id,name,type,status,md5,server_md5,download_md5,is_merged,is_deleted from CTE `
	rows := m.Tx.MustQuery(query+where+" order by n desc", args...)
	defer rows.Close()
	result := []models.File{}
	for rows.Next() {
		file := models.File{}
		rows.MustScan(&file.Id, &file.Name, &file.File_type, &file.Status, &file.Md5, &file.Server_md5, &file.Download_md5, &file.Is_merged, &file.Is_deleted)
		result = append(result, file)
	}
	return result
}
func (m *SQLITEManager) GetFile(id string) *models.File {
	files := m.getFiles("where id=$1", id)
	if len(files) == 1 {
		return &files[0]
	} else {
		return nil
	}
}

func (m *SQLITEManager) GetFiles() []models.File {
	return m.getFiles("")
}

func (m *SQLITEManager) GetMergedFiles() []models.File {
	return m.getFiles("where is_merged=1")
}
func (m *SQLITEManager) GetFileByPath(path string) *models.File {
	sql := `
	WITH RECURSIVE cte as(
		select name as path,* from file where p_id is null
		union all
		select cte.path || '/' || file.name as p,file.* from file inner join cte on file.p_id=cte.id where $1 like (p||'%')
	)
	select id from cte where path=$1 and is_deleted is not 1`
	id := ""
	row := m.Tx.MustQueryRow(sql, path)
	if err := row.Scan(&id); err != nil {
		if err == dbsqql.ErrNoRows {
			return nil
		} else {
			panic(err)
		}
	}
	return m.GetFile(id)
}
