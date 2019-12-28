package x

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func HideFile(path string) error {
	if !PathExist(path) {
		return errors.New("file or directory not exits")
	}
	if !strings.HasPrefix(filepath.Base(path), ".") {
		err := os.Rename(path, "."+path)
		if err != nil {
			return err
		}
	}
	return nil
}

func IsHidden(path string) (bool, error) {
	if !PathExist(path) {
		return false, errors.New("file or directory not exits")
	}
	return strings.HasPrefix(filepath.Base(path), "."), nil
}
