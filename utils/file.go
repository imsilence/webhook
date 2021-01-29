package utils

import (
	"io"
	"os"
	"path/filepath"
)

func Mkdir(path string) error {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return os.MkdirAll(path, os.ModePerm)
		}
		return err
	}
	return nil
}

func FileExists(path string) bool {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false
		}
		return true
	}
	return true
}

func CopyFile(src, dst string) error {

	if err := Mkdir(filepath.Dir(dst)); err != nil {
		return err
	}

	sFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sFile.Close()

	dFile, err := os.Create(dst + ".bak")
	if err != nil {
		return err
	}
	defer dFile.Close()

	if _, err := io.Copy(dFile, sFile); err != nil {
		return err
	}
	return os.Rename(dst+".bak", dst)
}
