package x

import (
	"bytes"
	"crypto/md5"
	"encoding/gob"
	"encoding/hex"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"os"

	"github.com/swishcloud/filesync/internal"
)

const Server_id = "5f1edf5340ad0252d2fb3f85519badb8"

func Encode(data interface{}) ([]byte, error) {
	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	err := enc.Encode(data)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
func Write(w io.Writer, data interface{}) error {
	b, err := Encode(data)
	if err != nil {
		return err
	}
	n, err := io.Copy(w, bytes.NewReader(b))
	log.Println("number of copied bytes:", n)
	if err != nil {
		return err
	}
	return nil
}
func Read(r io.Reader, data interface{}) error {
	dec := gob.NewDecoder(r)
	err := dec.Decode(data)
	if err != nil {
		return err
	}
	return nil
}
func ReadBytes(r io.Reader, size int) (int, []byte, error) {
	buf := bytes.NewBuffer([]byte{})
	w, err := io.CopyN(buf, r, int64(size))
	if err != nil {
		return 0, nil, err
	}
	if int(w) != size {
		panic(errors.New("size error"))
	}
	return int(w), buf.Bytes(), nil
}

func Hash_file_md5(filePath string) (string, error) {
	var returnMD5String string
	file, err := os.Open(filePath)
	if err != nil {
		return returnMD5String, err
	}
	defer file.Close()
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return returnMD5String, err
	}
	hashInBytes := hash.Sum(nil)
	returnMD5String = hex.EncodeToString(hashInBytes)
	return returnMD5String, nil

}

func PathExist(file_path string) bool {
	_, err := os.Stat(file_path)
	return err == nil
}

func GetApiUrlPath(p string) string {
	return internal.GlobalConfig().BaseApiUrlPath + p
}
func LogReader(prefix string, r io.Reader) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		panic(err)
	}
	log.Println(prefix, string(b))
}
