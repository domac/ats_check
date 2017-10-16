package util

import (
	"os/exec"
	"strconv"
	"strings"
)

//脚本执行工具

//直接执行命令脚本
func ShellRun(script string) error {
	return exec.Command("sh", "-c", script).Run()
}

//直接执行命令脚本，返回 []byte.
func Output(script string) ([]byte, error) {
	return exec.Command("sh", "-c", script).Output()
}

//直接执行命令脚本，返回 string.
func String(script string) (result string, err error) {
	bs, err := Output(script)
	if err != nil {
		return
	}
	result = string(bs)
	return
}

//直接执行命令脚本，返回 int.
func Int(script string) (result int, err error) {
	s, err := String(script)
	if err != nil {
		return
	}
	s = strings.TrimSpace(s)
	result, err = strconv.Atoi(s)
	return
}
