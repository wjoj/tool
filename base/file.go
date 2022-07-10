package base

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"gopkg.in/yaml.v2"
)

func FileOpenAppend(name string) (*os.File, error) {
	return os.OpenFile(name, os.O_RDWR|os.O_APPEND, os.ModePerm)
}

func IsDir(p string) (bool, error) {
	info, err := os.Stat(p)
	if err != nil {
		return false, err
	}
	if info != nil && !info.IsDir() {
		return false, nil
	}
	return true, nil
}

func IsFile(fpath string) bool {
	_, err := os.Stat(fpath)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}

func IsYaml(fpath string) bool {
	ext := filepath.Ext(fpath)
	ext = strings.ToLower(ext)
	switch ext {
	case ".yaml", ".yml", "yaml", "yml":
		return true
	}
	return false
}

func ReadYaml(fpath string, v any) error {
	if !IsYaml(fpath) {
		return errors.New("the file suffix is not `.yaml `")
	}
	yamlFile, err := ioutil.ReadFile(fpath)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(yamlFile, v)
	if err != nil {
		return fmt.Errorf("error reading yaml file, %v", err)
	}
	return nil
}

func ReadYamls(fpath string, vs ...any) error {
	if !IsYaml(fpath) {
		return errors.New("the file suffix is not `.yaml `")
	}
	yamlFile, err := ioutil.ReadFile(fpath)
	if err != nil {
		return err
	}
	for _, v := range vs {
		err = yaml.Unmarshal(yamlFile, v)
		if err != nil {
			return fmt.Errorf("error reading yaml file(%s), %v", reflect.TypeOf(v).Name(), err)
		}
	}

	return nil
}
