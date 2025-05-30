package utils

import "os"

func FileOpenAppend(name string) (*os.File, error) {
	return os.OpenFile(name, os.O_RDWR|os.O_APPEND, os.ModePerm)
}

func FileRead(path string) ([]byte, error) {
	return os.ReadFile(path)
}
