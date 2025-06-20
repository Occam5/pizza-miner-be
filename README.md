# Singo

Singo: Simple Single Golang Web Service

go-crud 正式改名为 Singo!

使用 Singo 开发 Web 服务: 用最简单的架构，实现够用的框架，服务海量用户

https://github.com/Gourouting/singo

## 更新日志

1.已支持接口测试 2.已经支持 go1.20，请安装这个版本的 golang 使用本项目

## 视频实况教程

[让我们写个 G 站吧！Golang 全栈编程实况](https://space.bilibili.com/10/channel/detail?cid=78794)

## 使用 Singo 开发的项目实例

仿 B 站的 G 站：https://github.com/Gourouting/giligili

Singo 框架为移动端提供 Token 登录的案例: https://github.com/bydmm/singo-token-exmaple

## 目的

本项目采用了一系列 Golang 中比较流行的组件，可以以本项目为基础快速搭建 Restful Web API

## 特色

本项目已经整合了许多开发 API 所必要的组件：

1. [Gin](https://github.com/gin-gonic/gin): 轻量级 Web 框架，自称路由速度是 golang 最快的
2. [GORM](https://gorm.io/index.html): ORM 工具。本项目需要配合 Mysql 使用
3. [Gin-Session](https://github.com/gin-contrib/sessions): Gin 框架提供的 Session 操作工具
4. [Go-Redis](https://github.com/go-redis/redis): Golang Redis 客户端
5. [godotenv](https://github.com/joho/godotenv): 开发环境下的环境变量工具，方便使用环境变量
6. [Gin-Cors](https://github.com/gin-contrib/cors): Gin 框架提供的跨域中间件
7. [httpexpect](https://github.com/gavv/httpexpect): 接口测试工具
8. 自行实现了国际化 i18n 的一些基本功能
9. 本项目是使用基于 cookie 实现的 session 来保存登录状态的，如果需要可以自行修改为 token 验证

本项目已经预先实现了一些常用的代码方便参考和复用:

1. 创建了用户模型
2. 实现了`/api/v1/user/register`用户注册接口
3. 实现了`/api/v1/user/login`用户登录接口
4. 实现了`/api/v1/user/me`用户资料接口(需要登录后获取 session)
5. 实现了`/api/v1/user/logout`用户登出接口(需要登录后获取 session)

本项目已经预先创建了一系列文件夹划分出下列模块:

1. api 文件夹就是 MVC 框架的 controller，负责协调各部件完成任务
2. model 文件夹负责存储数据库模型和数据库操作相关的代码
3. service 负责处理比较复杂的业务，把业务代码模型化可以有效提高业务代码的质量（比如用户注册，充值，下单等）
4. serializer 储存通用的 json 模型，把 model 得到的数据库模型转换成 api 需要的 json 对象
5. cache 负责 redis 缓存相关的代码
6. auth 权限控制文件夹
7. util 一些通用的小工具
8. conf 放一些静态存放的配置文件，其中 locales 内放置翻译相关的配置文件

## Godotenv

项目在启动的时候依赖以下环境变量，但是在也可以在项目根目录创建.env 文件设置环境变量便于使用(建议开发环境使用)

```shell
MYSQL_DSN="db_user:db_password@/db_name?charset=utf8&parseTime=True&loc=Local" # Mysql连接地址
REDIS_ADDR="127.0.0.1:6379" # Redis端口和地址
REDIS_PW="" # Redis连接密码
REDIS_DB="" # Redis库从0到10
SESSION_SECRET="setOnProducation" # Seesion密钥，必须设置而且不要泄露
GIN_MODE="debug"
```

## Go Mod

本项目使用[Go Mod](https://github.com/golang/go/wiki/Modules)管理依赖。

```shell
go mod init singo
export GOPROXY=http://mirrors.aliyun.com/goproxy/
go run main.go // 自动安装
```

## 运行

```shell
go run main.go
```

项目运行后启动在 3001 端口（可以修改，参考 gin 文档)

## 接口测试

【新】本项目内置了接口测试的内容

#### 使用方法

0. 确保自己在项目根目录下
1. 在 test 目录下创建 test 专用的环境变量文件

```
cp test/.env.example test/.env
```

2. 修改`test/.env`文件内容里的环境变量，保证可以正常连接 mysql/redis
3. 在项目根目录执行测试，并开启`-v`检查下测试是否在正确运行

```
go test -v ./test
```

4. 确保测试可正确运行后，后续去掉-v 参数，查看测试是否通过

```
go test ./test
ok      singo/test      (cached)
```
