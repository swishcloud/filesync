package session

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/swishcloud/filesync/message"
	"github.com/swishcloud/filesync/x"
	"github.com/swishcloud/gostudy/common"
)

type Session struct {
	_c                     net.Conn
	written                int64
	read                   int64
	last_msg_rest          int64
	presentWriteProgress   PresentWriteProgress
	write_speed_counter    int64
	write_speed            int64
	write_speed_clear_time time.Time
	closed                 bool
}
type PresentWriteProgress func(n int)

func NewSession(c net.Conn) *Session {
	s := new(Session)
	s._c = c
	go s.speedTimer()
	return s
}
func (s *Session) speedTimer() {
	for !s.closed {
		time.Sleep(time.Second * 1)
		s.write_speed = s.write_speed_counter
		s.write_speed_counter = 0
		s.write_speed_clear_time = time.Now()
	}
}
func (s *Session) Close() {
	err := s._c.Close()
	if err != nil {
		log.Fatal(err)
	}
	s.closed = true
}
func (s *Session) ReadMessage() (*message.Message, error) {
	if s.last_msg_rest != 0 {
		return nil, errors.New("last read not completed")
	}
	var size_b []byte
	size_buf := new(bytes.Buffer)
	for {
		_, err := io.CopyN(size_buf, s, 1)
		if err != nil {
			return nil, err
		}
		size_b = size_buf.Bytes()
		if size_b[len(size_b)-1] == 0 {
			size_b = size_b[:len(size_b)-1]
			break
		}
	}
	size, err := strconv.Atoi(string(size_b))
	if err != nil {
		return nil, err
	}

	msg_buf := new(bytes.Buffer)
	_, err = io.CopyN(msg_buf, s, int64(size))
	if err != nil {
		return nil, err
	}
	msg_b := msg_buf.Bytes()
	msg, err := message.ReadMessage(bytes.NewReader(msg_b))
	if err != nil {
		return nil, err
	}
	s.last_msg_rest = msg.BodySize
	return msg, nil
}
func (s *Session) Write(p []byte) (n int, err error) {
	t := time.Now()
	n, err = s._c.Write(p)
	if s.presentWriteProgress != nil {
		s.presentWriteProgress(n)
	}
	s.written += int64(n)
	if !t.Before(s.write_speed_clear_time) {
		s.write_speed_counter += int64(n)
	}
	return n, err
}
func (s *Session) Read(p []byte) (n int, err error) {
	n, err = s._c.Read(p)
	s.read += int64(n)
	s.last_msg_rest -= int64(n)
	return n, err
}
func (s *Session) SendMessage(msg *message.Message, payload io.Reader, payload_size int64) error {
	if msg.MsgType == 0 {
		return errors.New("message type value must be non zero")
	}
	if payload == nil && payload_size != 0 {
		return errors.New("parameter error")
	}

	msg.BodySize = payload_size

	msg_b, err := x.Encode(msg)
	if err != nil {
		return err
	}

	size_b := []byte(strconv.Itoa(len(msg_b)))

	_, err = io.CopyN(s, bytes.NewReader(size_b), int64(len(size_b)))
	if err != nil {
		return err
	}

	_, err = io.CopyN(s, bytes.NewReader([]byte{0}), 1)
	if err != nil {
		return err
	}
	_, err = io.CopyN(s, bytes.NewReader(msg_b), int64(len(msg_b)))
	if err != nil {
		return err
	}
	if payload != nil && payload_size > 0 {
		written := int64(0)
		total := msg.BodySize
		s.presentWriteProgress = func(n int) {
			written += int64(n)
			percent := int(float64(written) / float64(total) * 100)
			s, u := common.FormatByteSize(s.write_speed)
			fmt.Print("\r")
			info := fmt.Sprintf("sent %d/%d bytes %d%%,%s %s/s               ", written, total, percent, s, u)
			info = common.StringLimitLen(info, 50)
			fmt.Print(info)
		}
		n, err := io.CopyN(s, payload, msg.BodySize)
		fmt.Println()
		s.presentWriteProgress = nil
		if err != nil {
			return err
		}
		if n != payload_size {
			return errors.New(fmt.Sprintf("unexpected error:payload size is %d bytes,but written %d bytes", payload_size, n))
		}
	}
	return nil
}
func (s *Session) Ack() error {
	msg := message.NewMessage(message.MT_ACK)
	return s.Send(msg, nil)
}
func (s *Session) SendFile(file_path string, pre_send func(filename string, md5 string, size int64) (offset int64, send bool)) error {
	msg := message.NewMessage(message.MT_FILE)
	md5, err := x.Hash_file_md5(file_path)
	if err != nil {
		return err
	}
	msg.Header["md5"] = md5
	f, err := os.Open(file_path)
	defer f.Close()
	if err != nil {
		return err
	}
	file_info, err := f.Stat()
	if err != nil {
		return err
	}
	msg.Header["file_name"] = file_info.Name()
	payload_size := file_info.Size()
	if pre_send != nil {
		offset, ok := pre_send(file_info.Name(), md5, file_info.Size())
		if !ok {
			return nil
		}
		_, err := f.Seek(offset, 1)
		if err != nil {
			return err
		}
		payload_size -= offset
	}
	return s.SendMessage(msg, f, payload_size)
}
func (s *Session) Send(msg *message.Message, data interface{}) error {
	var payload io.Reader = nil
	var payload_size = 0
	if data != nil {
		b, err := json.Marshal(data)
		if err != nil {
			return err
		}
		payload = bytes.NewReader(b)
		payload_size = len(b)
	}
	return s.SendMessage(msg, payload, int64(payload_size))
}
func (s *Session) Fetch(msg *message.Message, data interface{}) (*message.Message, error) {
	err := s.Send(msg, data)
	if err != nil {
		return nil, err
	}
	return s.ReadMessage()
}
func (s *Session) ReadJson(size int, v interface{}) error {
	buf := new(bytes.Buffer)
	_, err := io.CopyN(buf, s, int64(size))
	if err != nil {
		return err
	}
	return json.Unmarshal(buf.Bytes(), v)
}
func (s *Session) ReadFile(filepath string, md5 string, size int64) (written int64, err error) {
	f, err := os.Create(filepath)
	if err != nil {
		return 0, err
	}
	written, err = io.CopyN(f, s, size)
	f.Close()
	hash, err := x.Hash_file_md5(filepath)
	if hash != md5 {
		return written, errors.New("md5 is inconsistent")
	}
	return written, err
}
func (s *Session) String() string {
	return fmt.Sprintf("remote_addr:%s", s._c.RemoteAddr())
}
