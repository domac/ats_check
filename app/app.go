package app

import (
	"github.com/domac/ats_check/log"
	"github.com/domac/ats_check/util"
	"os"
	"path/filepath"
)

var started bool

//应用启动
func Startup(ats_config_dir, log_path string) (err error) {
	if started {
		return
	}
	dir := filepath.Dir(log_path)
	util.ShellRun("mkdir -p " + dir)
	util.ShellRun("touch  " + log_path)
	//初始化日志
	log.LogInit(log_path, "info")

	if !util.FileExists(ats_config_dir) {
		log.GetLogger().Errorln("ats config dir not exist:", ats_config_dir)
		os.Exit(2)
	}

	parents_config := filepath.Join(ats_config_dir, "parents.config")

	go parents_check(parents_config)

	log.GetLogger().Infoln("服务初始化")
	return
}

//上层节点检测
func parents_check(parents_config string) {
	log.GetLogger().Infof("检测配置文件:%s", parents_config)
}

//停止服务
func Shutdown(i interface{}) {
	println()
	log.GetLogger().Infoln("application ready to shutdown")
}
