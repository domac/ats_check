package app

import (
	"errors"
	"fmt"
	"github.com/domac/ats_check/log"
	"github.com/domac/ats_check/util"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var started bool

var ErrServerDown = errors.New("server down")

var CURRENT_PARENTS = map[string]string{}

func init() {
	rand.Seed(time.Now().UnixNano())
}

type App struct {
	exitChan chan int
	cfg      *AppConfig
	sync.Mutex
}

//创建后台进程对象
func NewApp(cfg *AppConfig, log_path string) *App {
	dir := filepath.Dir(log_path)
	util.ShellRun("mkdir -p " + dir)
	util.ShellRun("touch  " + log_path)
	//初始化日志
	log.LogInit(log_path, "info")
	a := &App{
		cfg:      cfg,
		exitChan: make(chan int),
	}
	return a
}

func (self *App) GetConfig() *AppConfig {
	return self.cfg
}

//应用启动
func (self *App) Startup() (err error) {
	log.GetLogger().Infoln("服务初始化")
	//上层节点检测
	self.parents_check()
	return
}

//上层节点检测
func (self *App) parents_check() {

	parent_config := self.cfg.Parents_config_path
	parents := self.cfg.Parents

	log.GetLogger().Infof("parent config path:%s", parent_config)

	//父节点信息不存在
	if len(parents) == 0 {
		log.GetLogger().Errorln("parents not exists!")
		self.Shutdown(nil)
		os.Exit(2)
	}

	for _, phost := range parents {
		go self.single_parent_check(phost)
	}
}

func (self *App) single_parent_check(phost string) {
	health_check_url := self.cfg.Health_check
	health_check_url = strings.Replace(health_check_url, "{parent}", phost, 1)
	log.GetLogger().Infof("parent check url: %s", health_check_url)
	CURRENT_PARENTS[phost] = health_check_url
	//调度定时器
	ticker := time.Tick(time.Duration(self.cfg.Check_duration_second) * time.Second)

	httpclient := util.NewFastHttpClient(500 * time.Millisecond)
	for {
		select {
		case <-ticker:
			//主体测试功能
			log.GetLogger().Infof("parent check now -> %s", phost)
			err := retry(self.cfg.Retry, time.Duration(self.cfg.Retry_sleep_ms)*time.Millisecond, func() error {
				statusCode, body, err := httpclient.Get(nil, health_check_url)
				//statusCode, body, err := httpclient.Get(nil, "http://localhost:10200/ats")
				body = body[:0]
				if err != nil {
					return err
				}
				//5xx 错误
				if statusCode >= 500 {
					return ErrServerDown
				}
				return nil
			})

			//重试完成后校验
			if err != nil {
				self.failover(phost)
				goto exit
			}
		case <-self.exitChan:
			goto exit
		}
	}
exit:
	log.GetLogger().Infof("%s check exit", phost)
	return
}

func (self *App) failover(phost string) {
	//数次重试失败,当前检测退出
	self.Lock()
	self.checkFailCallBack(phost)
	self.Unlock()

	//全部PARENTS已被清除的情况
	if len(CURRENT_PARENTS) == 0 {
		self.recoverRemap()
	}
	//重新加载
	reload()
	log.GetLogger().Printf("%s failover done", phost)
}

func (self *App) checkFailCallBack(phost string) error {
	log.GetLogger().Infof("%s health check callback {%s}", phost, self.cfg.Parents_config_path)

	buf, err := ioutil.ReadFile(self.cfg.Parents_config_path)
	if err != nil {
		return err
	}

	content := string(buf)

	//替换
	newContent := strings.Replace(content, phost+";", "", 100)
	newContent = strings.Replace(newContent, phost, "", 100)

	//重新写入
	err = ioutil.WriteFile(self.cfg.Parents_config_path, []byte(newContent), 0777)
	if err != nil {
		return err
	}
	delete(CURRENT_PARENTS, phost)
	return nil
}

func (self *App) recoverRemap() {

	//备份原文件
	remap_file := self.cfg.Remap_config_path
	bf := util.BackupFile(remap_file)

	if bf == "" {
		log.GetLogger().Errorf("backup fail: %s", bf)
	}

	//替换文件
	dir := filepath.Dir(self.cfg.filepath)
	targetFile := filepath.Join(dir, "remap.config")
	cmd := fmt.Sprintf("cp -r %s %s", targetFile, bf)
	util.ShellRun(cmd)
}

func reload() error {
	cmd := "sudo sh /apps/sh/ats.sh reload"
	res, err := util.String(cmd)
	log.GetLogger().Infof("reload cmd : %s", cmd)

	if err != nil {
		log.GetLogger().Errorf("reload error: %s", err.Error())
		return err
	}
	log.GetLogger().Infof("reload result : %s", res)
	return nil
}

//重试函数
func retry(attempts int, sleep time.Duration, f func() error) error {
	if err := f(); err != nil {
		if attempts--; attempts > 0 {
			time.Sleep(sleep)
			log.GetLogger().Infoln("try to make connection with parent")
			return retry(attempts, sleep, f)
		}
		log.GetLogger().Infof("retry finish")
		return err
	}
	return nil
}

//停止服务
func (self *App) Shutdown(i interface{}) {
	println()
	log.GetLogger().Infoln("application ready to stop")
	close(self.exitChan)
	log.GetLogger().Infoln("application shutdown !!!")
}
