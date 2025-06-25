package main

import (
	"singo/conf"
	"singo/model"
	"singo/server"
	"singo/service"
)

func main() {
	// 从配置文件读取配置
	conf.Init()

	// 初始化数据库
	model.Database(conf.DatabaseConfig())

	// 执行数据库迁移
	model.Migration()

	// 初始化所有活跃奖池的大奖更新器
	service.GetPrizeUpdaterService().InitializeUpdaters()

	// 装载路由
	r := server.NewRouter()

	// 启动饥饿值更新工作器
	service.GetWebSocketManager().StartHungerUpdateWorker()

	// 运行服务器
	r.Run(":3001")
}
