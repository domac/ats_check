package main

import (
	"flag"
	"fmt"
	"github.com/domac/ats_check/app"
	"github.com/domac/ats_check/log"
	"github.com/domac/ats_check/web"
	l "log"
	_ "net/http/pprof"
)

var (
	port           = flag.String("port", "10200", "server port")
	log_path       = flag.String("log", "/apps/logs/ats_check/ats_check.log", "log path")
	ats_config_dir = flag.String("config_dir", "/apps/conf/trafficserver", "ats config files dir")
)

//prof command:
//go tool pprof --seconds 50 http://localhost:10200/debug/pprof/profile
func main() {
	println(app.Version)
	flag.Parse()

	if err := app.Startup(*ats_config_dir, *log_path); err != nil {
		l.Fatal(err)
		return
	}

	addr := fmt.Sprintf("localhost:%s", *port)
	log.GetLogger().Infof("ats_check 服务 http://%s/ats/parents", addr)
	httpServer, err := web.InitServer(addr)
	if err != nil {
		log.GetLogger().Error(err)
		return
	}

	go func() {
		err := httpServer.ListenAndServe()
		if err != nil {
			panic(err.Error())
		}
	}()

	//注册退出事件
	app.On(app.EXIT, app.Shutdown)
	app.Wait()
	app.Emit(app.EXIT, nil)
	log.GetLogger().Infoln("ats_check is exit now !")
}
