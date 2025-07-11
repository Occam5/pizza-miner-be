package conf

import (
	"os"
	"singo/cache"
	"singo/model"
	"singo/util"

	"github.com/joho/godotenv"
)

// Init 初始化配置项
func Init() {
	// 从本地读取环境变量
	godotenv.Load()

	// 设置日志级别
	util.BuildLogger(os.Getenv("LOG_LEVEL"))

	// 读取翻译文件
	if err := LoadLocales("conf/locales/zh-cn.yaml"); err != nil {
		util.Log().Panic("翻译文件加载失败", err)
	}

	// 连接数据库
	model.Database(DatabaseConfig())
	cache.Redis()
}

// DatabaseConfig 获取数据库配置
func DatabaseConfig() string {
	return os.Getenv("MYSQL_DSN")
}
