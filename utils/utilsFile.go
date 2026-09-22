package utils

import (
	"io"
	"os"
)

func CopyFile(fromFileName, toFileName string) error {
	var err error
	var r *os.File
	if r, err = os.Open(fromFileName); err != nil {
		return err
	}
	defer r.Close()
	fi, err := r.Stat()
	if err != nil {
		return err
	}
	tmpFileName := toFileName + ".tmp"
	var w *os.File
	if w, err = os.Create(tmpFileName); err != nil {
		return err
	}
	_, copyErr := io.Copy(w, r)
	if closeErr := w.Close(); copyErr == nil {
		copyErr = closeErr
	}
	if copyErr == nil {
		copyErr = os.Chmod(tmpFileName, fi.Mode().Perm())
	}
	if copyErr == nil {
		copyErr = os.Rename(tmpFileName, toFileName)
	} else {
		_ = os.Remove(tmpFileName)
	}
	return copyErr
}

func FileExists(fileName string) bool {
	if _, err := os.Stat(fileName); err == nil {
		return true
	}
	return false
}
