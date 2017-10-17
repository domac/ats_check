package app

import (
	"errors"
	"fmt"
	"github.com/domac/ats_check/log"
	"github.com/domac/ats_check/util"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

//服务器down掉异常
var ErrServerDown = errors.New("parent server down")

//上层节点结构
type ParentServer struct {
	Host     string
	Working  bool
	MarkDown bool
}

type App struct {
	exitChan chan int
	cfg      *AppConfig
	parents  map[string]*ParentServer
	sync.Mutex
	parentIsProxy bool
}

//创建后台进程对象
func NewApp(cfg *AppConfig, log_path string) *App {
	dir := filepath.Dir(log_path)
	util.ShellRun("mkdir -p " + dir)
	util.ShellRun("touch  " + log_path)
	//初始化日志
	log.LogInit(log_path, "info")
	a := &App{
		cfg:           cfg,
		exitChan:      make(chan int),
		parents:       make(map[string]*ParentServer),
		parentIsProxy: true,
	}
	return a
}

//应用启动
//上层节点检测
func (self *App) Startup() (err error) {

	log.GetLogger().Infoln("服务初始化")

	parent_config := self.cfg.Parents_config_path
	parents := self.cfg.Parents

	log.GetLogger().Infof("上层节点的ATS配置文件:%s", parent_config)

	//父节点信息不存在, 则退出
	if len(parents) == 0 {
		log.GetLogger().Errorln("上层结构不存在，请检查配置文件是否填写!")
		self.Shutdown(nil)
		os.Exit(2)
	}
	for _, phost := range parents {
		go self.parentHealthCheck(phost)
	}
	return
}

//上层节点健康检查
func (self *App) parentHealthCheck(phost string) {

	//初始化上层节点状态
	self.updateParent(phost, true)

	health_check_url := self.cfg.Health_check
	health_check_url = strings.Replace(health_check_url, "{parent}", phost, 1)
	log.GetLogger().Infof("上层节点健康检查URL: %s", health_check_url)
	//调度定时器
	ticker := time.Tick(time.Duration(self.cfg.Check_duration_second) * time.Second)

	httpclient := util.NewFastHttpClient(500 * time.Millisecond)
	for {
		select {
		case <-ticker:

			if httpclient == nil {
				log.GetLogger().Infoln("重建httpclient")
				httpclient = util.NewFastHttpClient(500 * time.Millisecond)
			}

			//主体测试功能
			log.GetLogger().Infof("定时健康监测 -> %s", phost)
			err := httpRetry(self.cfg.Retry, time.Duration(self.cfg.Retry_sleep_ms)*time.Millisecond, func() error {
				statusCode, body, err := httpclient.Get(nil, health_check_url)
				body = body[:0] //清空body
				if err != nil {
					return err
				}
				//出现 5xx 错误
				if statusCode >= 500 {
					return ErrServerDown
				}
				return nil
			})

			self.updateParent(phost, err == nil)

			//故障恢复
			self.failover(phost)

		case <-self.exitChan:
			goto exit
		}
	}
exit:
	log.GetLogger().Infof("%s 健康监测功能退出", phost)
	return
}

//故障处理
func (self *App) failover(phost string) {
	if parentServer, ok := self.parents[phost]; ok {
		if parentServer.MarkDown && parentServer.Working {
			//服务恢复正常的情况
			self.backwardRecover(parentServer)
		} else if !parentServer.MarkDown && !parentServer.Working {
			//服务出现不可用的情况
			self.forwardRecover(parentServer)
		}

		// if !parentServer.Working {
		// 	self.forwardRecover(parentServer)
		// } else {
		// 	self.parentIsProxy = true
		// 	parentServer.MarkDown = false
		// }
	}
}

//----------------------- 正向处理(容错处理) -----------------------//
//服务出现不可用的情况
func (self *App) forwardRecover(parentServer *ParentServer) {
	log.GetLogger().Infof("%s >>>>>>>> forward recover", parentServer.Host)
	defer func() {
		parentServer.MarkDown = true //markdown处理
	}()

	if !parentServer.MarkDown {
		self.updateParentConfig()
		//全部上层节点已经不可用
		if self.parentIsProxy && len(self.getNotWorkingParentsHosts()) == len(self.GetParentsHosts()) {
			self.forwardRecordsConfig()
			self.forwardRemapConfig()
			self.parentIsProxy = false //关闭父代理功能
		}
		self.reloadConfig()
	}
}

//records.config 关闭parent proxy功能
func (self *App) forwardRecordsConfig() {
	field := "0"
	testCmd := `sed -i 's/CONFIG[ ][ ]*proxy.config.http.parent_proxy_routing_enable[ ][ ]*INT[ ][ ]*.*/CONFIG proxy.config.http.parent_proxy_routing_enable INT %s/g' %s`
	cmd := fmt.Sprintf(testCmd, field, self.cfg.Records_config_path)
	log.GetLogger().Infof("update forward records config command: %s", cmd)
	util.ShellRun(cmd)
}

//remap.config 配置访问源站
func (self *App) forwardRemapConfig() {
	//备份原文件
	bf := util.BackupFile(self.cfg.Remap_config_path)
	if bf != "" {
		log.GetLogger().Info("备份remap成功")
		//替换文件
		dir := filepath.Dir(self.cfg.filepath)
		sourceFile := filepath.Join(dir, "remap_parent.config")
		cmd := fmt.Sprintf("cp  %s %s", sourceFile, self.cfg.Remap_config_path)
		util.ShellRun(cmd)
	}
}

//----------------------- 反向处理(恢复处理) -----------------------//

//服务恢复正常的情况
func (self *App) backwardRecover(parentServer *ParentServer) {
	log.GetLogger().Infof("%s backward recover", parentServer.Host)
	self.Lock()
	defer func() {
		parentServer.MarkDown = false //markdown处理
		self.Unlock()
	}()

	if parentServer.MarkDown {
		self.updateParentConfig()

		//之前的状态是回源站
		if !self.parentIsProxy {
			self.backwardRemapConfig()
			self.backwardRecordsConfig()
			self.parentIsProxy = true
		}
		self.reloadConfig()
	}

}

//records.config 关闭parent proxy功能
func (self *App) backwardRecordsConfig() {
	field := "1"
	testCmd := `sed -i 's/CONFIG[ ][ ]*proxy.config.http.parent_proxy_routing_enable[ ][ ]*INT[ ][ ]*.*/CONFIG proxy.config.http.parent_proxy_routing_enable INT %s/g' %s`
	cmd := fmt.Sprintf(testCmd, field, self.cfg.Records_config_path)
	util.ShellRun(cmd)
}

func (self *App) backwardRemapConfig() {
	dir := filepath.Dir(self.cfg.filepath)
	sourceFile := filepath.Join(dir, "remap_edge.config")
	cmd := fmt.Sprintf("cp  %s %s", sourceFile, self.cfg.Remap_config_path)
	util.ShellRun(cmd)
}

//----------------------- 公共方法 -----------------------//

//重试函数
func httpRetry(attempts int, sleep time.Duration, f func() error) error {
	if err := f(); err != nil {
		if attempts--; attempts > 0 {
			time.Sleep(sleep)
			log.GetLogger().Infoln("上层节点连接重试")
			return httpRetry(attempts, sleep, f)
		}
		log.GetLogger().Infof("重试终止，上层节点不可用")
		return err
	}
	return nil
}

//更新parent.config信息
func (self *App) updateParentConfig() {
	pws := strings.Join(self.getWorkingParentsHosts(), ";")

	testCmd := `sed -i 's/[^#].*parent=".*/dest_domain=. method=get  parent="%s" round_robin=consistent_hash/g' %s`
	cmd := fmt.Sprintf(testCmd, pws, self.cfg.Parents_config_path)
	log.GetLogger().Infof("update parent config command: %s", cmd)
	util.ShellRun(cmd)
}

//更新父节点信息
func (self *App) updateParent(phost string, working bool) {
	self.Lock()
	defer self.Unlock()
	if p, ok := self.parents[phost]; ok {
		p.Working = working
		self.parents[phost] = p
	} else {
		np := &ParentServer{}
		np.Host = phost
		np.Working = working
		self.parents[phost] = np
	}
}

//获取所有上层节点列表
func (self *App) GetParentsHosts() []string {
	contents := []string{}
	for _, ps := range self.parents {
		contents = append(contents, ps.Host)
	}
	return contents
}

//获取正常工作的上层节点列表
func (self *App) getWorkingParentsHosts() []string {
	self.Lock()
	defer self.Unlock()
	contents := []string{}
	for _, ps := range self.parents {
		if ps.Working {
			contents = append(contents, ps.Host)
		}
	}
	return contents
}

//获取挂掉的工作的上层节点列表
func (self *App) getNotWorkingParentsHosts() []string {
	self.Lock()
	defer self.Unlock()
	contents := []string{}
	for _, ps := range self.parents {
		if !ps.Working {
			contents = append(contents, ps.Host)
		}
	}
	return contents
}

func (self *App) reloadConfig() {
	cmd := "sh /apps/sh/ats.sh reload"
	res, err := util.String(cmd)
	if err != nil {
		log.GetLogger().Error(err)
	}
	log.GetLogger().Infof("reload result: %s", res)
}

//停止服务
func (self *App) Shutdown(i interface{}) {
	close(self.exitChan)
	log.GetLogger().Infoln("应用服务关闭!!!")
}
