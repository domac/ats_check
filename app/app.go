package app

import (
	"errors"
	"fmt"
	"github.com/domac/ats_check/log"
	"github.com/domac/ats_check/util"
	"net"
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

//HaProxy节点结构
type HaProxyServer struct {
	Host     string
	Working  bool
	MarkDown bool
}

type App struct {
	exitChan chan int
	cfg      *AppConfig
	parents  map[string]*ParentServer
	haproxys map[string]*HaProxyServer
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
		haproxys:      make(map[string]*HaProxyServer),
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
	haproxys := self.cfg.Haproxys

	log.GetLogger().Infof("上层节点的ATS配置文件:%s", parent_config)
	//父节点信息不存在, 则退出
	if len(parents) == 0 {
		log.GetLogger().Errorln("上层结构不存在，请检查配置文件是否填写!")
		self.Shutdown(nil)
		os.Exit(2)
	}

	if self.cfg.Is_parent == 1 {
		//parent节点模式
		for _, hhost := range haproxys {
			go self.haproxyHealthCheck(hhost)
		}
	} else {
		//边缘节点模式
		for _, phost := range parents {
			go self.parentHealthCheck(phost)
		}
	}
	return
}

//Haproxy节点的监控检测
func (self *App) haproxyHealthCheck(hhost string) {
	log.GetLogger().Infof("Haproxy节点健康检查开始 : %s", hhost)
	//调度定时器
	ticker := time.Tick(time.Duration(self.cfg.Check_duration_second) * time.Second)
	for {
		select {
		case <-ticker:
			//主体测试功能
			log.GetLogger().Infof("HA定时健康监测 => %s", hhost)
			err := retry(self.cfg.Retry, time.Duration(self.cfg.Retry_sleep_ms)*time.Millisecond,
				func() error {
					return self.tcpPortCheck(hhost, 80)
				})
			self.updateHaproxy(hhost, err == nil)
			//HA故障恢复
			self.haFailover(hhost)
		case <-self.exitChan:
			goto exit
		}
	}
exit:
	log.GetLogger().Infof("HA %s 健康监测功能退出", hhost)
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
			err := retry(self.cfg.Retry, time.Duration(self.cfg.Retry_sleep_ms)*time.Millisecond, func() error {
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

//故障处理：Ha配置
func (self *App) haFailover(hhost string) {
	if haServer, ok := self.haproxys[hhost]; ok {
		if !haServer.Working {
			self.forwardHaproxyRecover(haServer)
		} else {
			haServer.MarkDown = false
		}
	}
}

//故障处理
func (self *App) failover(phost string) {
	if parentServer, ok := self.parents[phost]; ok {
		// if parentServer.MarkDown && parentServer.Working {
		// 	//服务恢复正常的情况
		// 	//self.backwardRecover(parentServer)
		// } else if !parentServer.MarkDown && !parentServer.Working {
		// 	//服务出现不可用的情况
		// 	self.forwardRecover(parentServer)
		// }

		if !parentServer.Working {
			self.forwardRecover(parentServer)
		} else {
			self.parentIsProxy = true
			parentServer.MarkDown = false
		}
	}
}

//----------------------- 正向处理(容错处理) -----------------------//

func (self *App) forwardHaproxyRecover(haproxyServer *HaProxyServer) {
	log.GetLogger().Infof("%s >>>>>>>> forward Haproxy recover", haproxyServer.Host)
	defer func() {
		haproxyServer.MarkDown = true //markdown处理
	}()

	if !haproxyServer.MarkDown {
		self.updateRemapHaproxyConfig(haproxyServer.Host)
		self.reloadConfig()
	}
}

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
		}
		self.reloadConfig()
	}

}

//records.config 关闭parent proxy功能
func (self *App) backwardRecordsConfig() {
	field := "1"
	//cmd := strings.Replace(self.cfg.Setup_records_config_cmd, "{PARENTS_ENBALE}", field, 1)
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
func retry(attempts int, sleep time.Duration, f func() error) error {
	if err := f(); err != nil {
		if attempts--; attempts > 0 {
			time.Sleep(sleep)
			log.GetLogger().Infoln("节点连接重试")
			return retry(attempts, sleep, f)
		}
		log.GetLogger().Infof("重试终止，节点不可用")
		return err
	}
	return nil
}

//tcp检测
func (self *App) tcpPortCheck(host string, port uint32) error {
	addr := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		log.GetLogger().Errorf("tcp connect to %s fail !", host)
		return err
	}
	defer conn.Close()
	return nil
}

//更新remap.config信息
func (self *App) updateRemapHaproxyConfig(hhost string) {
	self.Lock()
	defer self.Unlock()
	log.GetLogger().Infof("update remap proxy config by host: ", hhost)
	testCmd := `sed -i 's/@pparam=%s//g' %s`
	cmd := fmt.Sprintf(testCmd, hhost, self.cfg.Remap_config_path)
	log.GetLogger().Infof("update parent config command: %s", cmd)
	util.ShellRun(cmd)
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

//更新Ha节点信息
func (self *App) updateHaproxy(hhost string, working bool) {
	self.Lock()
	defer self.Unlock()
	if p, ok := self.haproxys[hhost]; ok {
		p.Working = working
		self.haproxys[hhost] = p
	} else {
		np := &HaProxyServer{}
		np.Host = hhost
		np.Working = working
		self.haproxys[hhost] = np
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

//获取正常工作的Ha节点列表
func (self *App) getWorkingHaproxyHosts() []string {
	self.Lock()
	defer self.Unlock()
	contents := []string{}
	for _, hs := range self.haproxys {
		if hs.Working {
			contents = append(contents, hs.Host)
		}
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
