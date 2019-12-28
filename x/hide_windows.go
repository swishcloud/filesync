package x

import (
	"syscall"
)

func HideFile(filename string) error {
	filenameW, err := syscall.UTF16PtrFromString(filename)
	if err != nil {
		return err
	}
	err = syscall.SetFileAttributes(filenameW, syscall.FILE_ATTRIBUTE_HIDDEN)
	if err != nil {
		return err
	}
	return nil
}
func IsHidden(path string) (bool, error) {
	p, e := syscall.UTF16PtrFromString(path)
	if e != nil {
		return false, e
	}
	attrs, e := syscall.GetFileAttributes(p)
	if e != nil {
		return false, e
	}
	return attrs&syscall.FILE_ATTRIBUTE_HIDDEN != 0, nil
}
