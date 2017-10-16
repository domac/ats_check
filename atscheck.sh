#!/usr/bin/env bash

APPLICATION=ats_check
LOG_PATH_FILE=/apps/logs/ats_check/ats_check.log

start(){
	nohup ${APPLICATION_PATH}${APPLICATION} --config=${APPLICATION_PATH}config/base.conf  2>&1 &
	sleep 2
	ps aux|grep ${APPLICATION}|grep base.conf|grep -v grep
	tail -n 18 ${LOG_PATH_FILE}
}

stop(){
	pid=$(ps aux|grep ${APPLICATION}|grep base.conf|grep -v grep|awk '{print $2}')
	if [[ $pid != "" ]]
	then
		ps aux|grep ${APPLICATION}|grep base.conf|grep -v grep|awk '{print $2}'|xargs kill -15
	fi
}

restart(){
	stop && start
}

status(){
	ps aux|grep ${APPLICATION}|grep -v grep
}

case $1 in
	"start")
		start
	;;

	"stop")
		stop
	;;

	"restart")
		restart
	;;
	
	"status")
		status
	;;
	
	*)
		echo "Usage:{start|stop|restart|status}"
esac