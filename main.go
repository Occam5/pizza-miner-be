package main

import (
	"singo/conf"
	"singo/server"
	"singo/service"
)

func main() {
	// 从配置文件读取配置
	conf.Init()

	// 装载路由
	r := server.NewRouter()

	// 启动饥饿值更新工作器
	service.GetWebSocketManager().StartHungerUpdateWorker()

	// 运行服务器
	r.Run(":3001")
}
