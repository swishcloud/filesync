package x

import (
	"bytes"
	"testing"
)

func Test_encode_decode(t *testing.T) {
	a := []string{"a", "b"}
	buf := bytes.NewBuffer(make([]byte, 0))
	err := Write(buf, a)
	if err != nil {
		t.Fatal(err)
	}
	b := new([]string)
	err = Read(buf, b)
	if err != nil {
		t.Fatal(err)
	}
	if (*b)[0] != "a" {
		t.FailNow()
	}
}
func Test_ReadBytes(t *testing.T) {
	greeting := "hello,world"
	g_b := []byte(greeting)
	_, b, err := ReadBytes(bytes.NewReader(g_b), len(g_b))
	str := string(b)
	println(str)
	if err != nil || greeting != str {
		t.FailNow()
	}

}
