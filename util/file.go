package util

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	WRITE_APPEND = 0
	WRITE_OVER   = 1
)

//------------------------------
//some tool for handling file
//------------------------------

func CheckDataFileExist(filePath string) error {

	if filePath == "" {
		return errors.New("数据文件路径为空")
	}

	if _, err := os.Stat(filePath); err != nil {
		return errors.New("PathError:" + err.Error())
	}
	return nil
}

func ReadLine(fileName string) ([]string, error) {
	if err := CheckDataFileExist(fileName); err != nil {
		return []string{}, err
	}

	f, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}

	result := []string{}

	buf := bufio.NewReader(f)
	for {
		line, err := buf.ReadString('\n')
		line = strings.TrimSpace(line)
		if err != nil {
			if err == io.EOF {
				return result, nil
			}
			return nil, err
		}
		if line != "" && !strings.HasPrefix(line, "#") {
			result = append(result, line)
		}

	}
	return result, nil
}

func RemoveDuplicatesAndEmpty(a []string) (ret []string) {
	a_len := len(a)
	for i := 0; i < a_len; i++ {
		if (i > 0 && a[i-1] == a[i]) || len(a[i]) == 0 {
			continue
		}
		ret = append(ret, a[i])
	}
	return
}

func RemoveFile(filepath string) error {
	err := os.Remove(filepath)
	if err != nil {
		return err
	}
	return nil
}

func CreateFile(filepath string) (string, error) {
	finfo, err := os.Stat(filepath)
	if err == nil {
		if finfo.IsDir() {
			return filepath, errors.New("filepath is a dir")
		} else {
			return filepath, errors.New("filepath exists")
		}
	}
	f, err := os.Create(filepath)
	if err != nil {
		fmt.Println(err.Error())
		return filepath, err
	}
	defer f.Close()
	return filepath, nil
}

func IsFile(filepath string) bool {
	finfo, err := os.Stat(filepath)
	if err == nil {
		if finfo.IsDir() {
			return false
		}
	} else {
		return false
	}
	return true
}

func openToAppend(fpath string) (*os.File, error) {
	f, err := os.OpenFile(fpath, os.O_RDWR|os.O_APPEND, 0777)
	if err != nil {
		f, err = os.Create(fpath)
		if err != nil {
			return f, err
		}
	}
	return f, nil
}

func openToOverwrite(fpath string) (*os.File, error) {
	f, err := os.OpenFile(fpath, os.O_RDWR|os.O_TRUNC, 0777)
	if err != nil {
		f, err = os.Create(fpath)
		if err != nil {
			return f, err
		}
	}
	return f, nil
}

func WriteIntoFile(filepath string, content []string, writeMode int) error {

	var f *os.File
	var err error

	if writeMode == WRITE_APPEND {
		f, err = openToAppend(filepath)
	} else {
		f, err = openToOverwrite(filepath)
	}

	if err != nil {
		return err
	}
	defer f.Close()

	for _, s := range content {
		fmt.Fprintln(f, s)
	}
	return nil
}

func WriteBytesIntoFile(filepath string, content []byte, writeMode int) error {

	var f *os.File
	var err error

	if writeMode == WRITE_APPEND {
		f, err = openToAppend(filepath)
	} else {
		f, err = openToOverwrite(filepath)
	}

	if err != nil {
		return err
	}
	defer f.Close()

	fmt.Fprintln(f, string(content))
	return nil
}

func SelfPath() string {
	path, _ := filepath.Abs(os.Args[0])
	return path
}

func SelfDir() string {
	return filepath.Dir(SelfPath())
}

func FileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func SearchFile(filename string, paths ...string) (fullpath string, err error) {
	for _, path := range paths {
		if fullpath = filepath.Join(path, filename); FileExists(fullpath) {
			return
		}
	}
	err = errors.New(fullpath + " not found in paths")
	return
}

func GrepFile(patten string, filename string) (lines []string, err error) {
	re, err := regexp.Compile(patten)
	if err != nil {
		return
	}

	fd, err := os.Open(filename)
	if err != nil {
		return
	}
	lines = make([]string, 0)
	reader := bufio.NewReader(fd)
	prefix := ""
	isLongLine := false
	for {
		byteLine, isPrefix, er := reader.ReadLine()
		if er != nil && er != io.EOF {
			return nil, er
		}
		if er == io.EOF {
			break
		}
		line := string(byteLine)
		if isPrefix {
			prefix += line
			continue
		} else {
			isLongLine = true
		}

		line = prefix + line
		if isLongLine {
			prefix = ""
		}
		if re.MatchString(line) {
			lines = append(lines, line)
		}
	}
	return lines, nil
}

func GetShortFile(fullName string) string {
	return filepath.Base(fullName)
}

func GetDir(fullName string) string {
	return filepath.Dir(fullName)
}

func GetFileExt(fullName string) string {
	return filepath.Ext(fullName)
}

func SearchFileInDir(dir string, file_name_part string) ([]string, error) {

	result := []string{}

	if !FileExists(dir) {
		return nil, errors.New("dir is not exist:" + dir)
	}

	if IsFile(dir) {
		return nil, errors.New("path must dir :" + dir)
	}

	files, _ := ioutil.ReadDir(dir)
	for _, file := range files {
		if file.IsDir() {
			continue
		} else {

			if file_name_part != "" {

				if strings.Contains(file.Name(), file_name_part) {
					result = append(result, file.Name())
				}

			} else {
				result = append(result, file.Name())
			}
		}
	}
	return result, nil
}
