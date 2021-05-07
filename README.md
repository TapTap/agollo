# Apollo - Go Client for Apollo

[![golang](https://img.shields.io/badge/Language-Go-green.svg?style=flat)](https://golang.org)
[![GoDoc](http://godoc.org/github.com/taptap/go-apollo?status.svg)](http://godoc.org/github.com/taptap/go-apollo)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

携程 Apollo Golang 版客户端

针对 [apollo openapi](https://github.com/shima-park/apollo-openapi) 的 golang 客户端封装

## 快速开始
### 获取安装
```
go get -u github.com/taptap/go-apollo
```

## Features
* 实时同步配置，配置改动监听
* 配置文件容灾
* 支持多 namespace, cluster
* 客户端 SLB
* 提供 Viper 配置库的 apollo 插件
* 支持通过 APOLLO_ACCESS_KEY 来实现 client 安全访问
* 支持自定义签名认证

## 示例

### 读取配置
此示例场景适用于程序启动时读取一次。不会额外启动 goroutine 同步配置
```
package main

import(
	"fmt"

	"github.com/taptap/go-apollo"
)

func main() {
	a, err := apollo.New("localhost:8080", "your_appid", apollo.AutoFetchOnCacheMiss())
	if err != nil {
		panic(err)
	}

	fmt.Println(
		// 默认读取 Namespace：application 下 key: foo 的 value
		a.Get("foo"),

		// 获取 namespace 为 test.json 的所有配置项
		a.GetNameSpace("test.json"),

		// 当 key：foo 不存在时，提供一个默认值 bar
		a.Get("foo", apollo.WithDefault("bar")),

		// 读取 Namespace 为 other_namespace, key: foo 的 value
		a.Get("foo", apollo.WithNamespace("other_namespace")),
	)
}
```

### 实时同步配置
启动一个 goroutine 实时同步配置，errorCh 返回 notifications/v2 非 httpcode(200) 的错误信息
```
a, err := apollo.New("localhost:8080", "your_appid", apollo.PreloadNamespaces("application", "test.json"))
//error handle...

errorCh := a.Start()  // Start 后会启动 goroutine 监听变化，并更新 apollo 对象内的配置 cache
// 或者忽略错误处理直接 a.Start()
```

### 配置监听
监听所有 namespace 配置变更事件
```
a, err := apollo.New("localhost:8080", "your_appid", apollo.PreloadNamespaces("application", "test.json"))
//error handle...

errorCh := a.Start()  // Start 后会启动 goroutine 监听变化，并更新 apollo 对象内的配置 cache
// 或者忽略错误处理直接 a.Start()

watchCh := a.Watch()

for {
	select {
	case err := <- errorCh:
		//handle error
	case resp := <-watchCh:
		fmt.Println(
			"Namespace:", resp.Namespace,
			"OldValue:", resp.OldValue,
			"NewValue:", resp.NewValue,
			"Error:", resp.Error,
		)
	}
}
```
### 配置文件容灾
初始化时增加 apollo.FailTolerantOnBackupExists() 即可，
在连接 apollo 失败时，如果在配置的目录下存在.apollo 备份配置，会读取备份在服务器无法连接的情况下
```
a, err := apollo.New("localhost:8080", "your_appid",
		apollo.FailTolerantOnBackupExists(),
		//other options...
	)
//error handle...
```

### 支持多 namespace
初始化时增加 apollo.AutoFetchOnCacheMiss() 当本地缓存中 namespace 不存在时，尝试去 apollo 缓存接口去获取
```
a, err := apollo.New("localhost:8080", "your_appid",
		apollo.AutoFetchOnCacheMiss(),
		//other options...
	)
//error handle...

appNS, aNS, bNS := a.GetNameSpace("application"), a.GetNameSpace("Namespace_A"), a.GetNameSpace("Namespace_B")

a.Get("foo") // 默认从 application 这个 namespace 中查找配置项
a.Get("foo", apollo.WithNamespace("Namespace_A")) // 从 Namespace_A 中获取配置项 foo
a.Get("foo", apollo.WithNamespace("Namespace_B")) // 从 Namespace_B 中获取配置项 foo
//...
```

或者初始化时增加 apollo.PreloadNamespaces("Namespace_A", "Namespace_B", ...) 预加载这几个 Namespace 的配置
```
a, err := apollo.New("localhost:8080", "your_appid",
		apollo.PreloadNamespaces("Namespace_A", "Namespace_B", ...),
		//other options...
	)
//error handle...
```

当然两者结合使用也是可以的。
```
a, err := apollo.New("localhost:8080", "your_appid",
		apollo.PreloadNamespaces("Namespace_A", "Namespace_B", ...),
		apollo.AutoFetchOnCacheMiss(),
		//other options...
	)
//error handle...
```

### 如何支持多 cluster
初始化时增加 apollo.Cluster("your_cluster")，并创建多个 Apollo 接口实例 [issue](https://github.com/taptap/go-apollo/issues/1)
```
cluster_a, err := apollo.New("localhost:8080", "your_appid",
		apollo.Cluster("cluster_a"),
		apollo.AutoFetchOnCacheMiss(),
		//other options...
	)

cluster_b, err := apollo.New("localhost:8080", "your_appid",
		apollo.Cluster("cluster_b"),
		apollo.AutoFetchOnCacheMiss(),
		//other options...
	)

cluster_a.Get("foo")
cluster_b.Get("foo")
//...
```

### 客户端 SLB
客户端通过 MetaServer 进行动态 SLB 的启用逻辑：

```
// 方式 1:
    // 使用者主动增加配置项 apollo.EnableSLB(true)
    a, err := apollo.New("localhost:8080", "your_appid", apollo.EnableSLB(true))


// 方式 2:
    // (客户端显示传递的 configServerURL) 和 (环境变量中的 APOLLO_CONFIGSERVICE) 都为空值
    //export APOLLO_CONFIGSERVICE=""
    // 此方式必须设置 export APOLLO_META="your meta_server address"
    a, err := apollo.New("","your_appid")
```

客户端静态 SLB(现在支持 "," 分割的多个 configServer 地址列表):

```
// 方式 1:
    // 直接传入 "," 分割的多个 configServer 地址列表
    a, err := apollo.New("localhost:8080,localhost:8081,localhost:8082", "your_appid")

// 方式 2:
    // 在环境变量中 APOLLO_CONFIGSERVICE 设置 "," 分割的多个 configServer 地址列表
    //export APOLLO_CONFIGSERVICE="localhost:8080,localhost:8081,localhost:8082"
    a, err := apollo.New("","your_appid")
```

SLB 更新间隔默认是 60s 和官方 java sdk 保持一致，可以通过 apollo.ConfigServerRefreshIntervalInSecond(time.Second * 90) 来修改
```
    a, err := apollo.New("localhost:8080", "your_appid",
        apollo.EnableSLB(true),
        apollo.ConfigServerRefreshIntervalInSecond(time.Second * 90),
    )
```

! SLB 的 MetaServer 地址来源 (用来调用接口获取 configServer 列表)，取下列表中非空的一项:
1. 用户显示传递的 configServerURL
2. 环境变量中的 APOLLO_META

! SLB 的默认采用的算法是 RoundRobin

### 初始化方式

三种 package 级别初始化，影响默认对象和 package 提供的静态方法。适用于不做对象传递，单一 AppID 的场景
```
// 读取当前目录下 app.properties，适用于原始 apollo 定义的读取固定配置文件同学
apollo.InitWithDefaultConfigFile(opts ...Option) error

apollo.Init(configServerURL, appID string, opts ...Option) (err error)

apollo.InitWithConfigFile(configFilePath string, opts ...Option) (err error)
```

两种新建对象初始化方法。返回独立的 Apollo 接口对象。互相之间不会影响，适用于多 AppID，Cluser, ConfigServer 配置读取
[issue](https://github.com/taptap/go-apollo/issues/1)
```
apollo.New(configServerURL, appID string, opts ...Option) (Apollo, error)
apollo.NewWithConfigFile(configFilePath string, opts ...Option) (Apollo, error)
```

### 初始化时可选配置项
更多配置请见 [options.go](https://github.com/taptap/go-apollo/blob/master/options.go)
```
        // 打印日志，打印日志注入有效的 io.Writer，默认: ioutil.Discard
	apollo.WithLogger(apollo.NewLogger(apollo.LoggerWriter(os.Stdout))),

	// 默认的集群名称，默认：default
	apollo.Cluster(cluster),

	// 预先加载的 namespace 列表，如果是通过配置启动，会在 app.properties 配置的基础上追加
	apollo.PreloadNamespaces("Namespace_A", "Namespace_B", ...),

	// 在配置未找到时，去 apollo 的带缓存的获取配置接口，获取配置
	apollo.AutoFetchOnCacheMiss(),

	// 备份文件存放地址，默认：当前目录下 /.apollo，一般结合 FailTolerantOnBackupExists 使用
	apollo.BackupFile("/tmp/xxx/.apollo")
	// 在连接 apollo 失败时，如果在配置的目录下存在.apollo 备份配置，会读取备份在服务器无法连接的情况下
	apollo.FailTolerantOnBackupExists(),
```

### 详细特性展示
请将 example/sample 下 app.properties 修改为你本地或者测试的 apollo 配置。
[示例代码](https://github.com/taptap/go-apollo/blob/master/examples/sample/main.go)

## 结合 viper 使用，提高配置读取舒适度
例如 apollo 中有以下配置:
```
appsalt = xxx
database.driver = mysql
database.host = localhost
database.port = 3306
database.timeout = 5s
//...
```

示例代码:
```
import(
    "fmt"
	"github.com/taptap/go-apollo/viper-remote"
	"github.com/spf13/viper"
)

type Config struct {
	AppSalt string         `mapstructure:"appsalt"`
	DB      DatabaseConfig `mapstructure:"database"`
}

type DatabaseConfig struct {
	Driver   string        `mapstructure:"driver"`
	Host     string        `mapstructure:"host"`
	Port     int           `mapstructure:"port"`
	Timeout time.Duration  `mapstructure:"timeout"`
	//...
}

func main(){
    remote.SetAppID("your_appid")
    v := viper.New()
    v.SetConfigType("prop") // 根据 namespace 实际格式设置对应 type
    err := v.AddRemoteProvider("apollo", "your_apollo_endpoint", "your_apollo_namespace")
    //error handle...
    err = v.ReadRemoteConfig()
    //error handle...

    // 直接反序列化到结构体中
    var conf Config
    err = v.Unmarshal(&conf)
    //error handle...
    fmt.Printf("%+v\n", conf)

    // 各种基础类型配置项读取
    fmt.Println("Host:", v.GetString("db.host"))
    fmt.Println("Port:", v.GetInt("db.port"))
    fmt.Println("Timeout:", v.GetDuration("db.timeout"))

    // 获取所有 key，所有配置
    fmt.Println("AllKeys", v.AllKeys(), "AllSettings",  v.AllSettings())
}
```

如果碰到 panic: codecgen version mismatch: current: 8, need 10 这种错误，详情请见 [issue](https://github.com/taptap/go-apollo/issues/14)
解决办法是将 etcd 升级到 3.3.13:
```
// 使用 go module 管理依赖包，使用如下命令更新到此版本，或者更高版本
go get github.com/coreos/etcd@v3.3.13+incompatible
```

### viper 配置同步
基于轮训的配置同步
```
    remote.SetAppID("your_appid")
    v := viper.New()
    v.SetConfigType("prop")
    err := v.AddRemoteProvider("apollo", "your_apollo_endpoint", "your_apollo_namespace")
    //error handle...
    err = v.ReadRemoteConfig()
    //error handle...

    for {
	time.Sleep(10 * time.Second)

	err := app.WatchRemoteConfig() // 每次调用该方法，会从 apollo 缓存接口获取一次配置，并更新 viper
	if err != nil {
		panic(err)
	}

	fmt.Println("app.AllSettings:", app.AllSettings())
     }
```
基于事件监听配置同步
```
    remote.SetAppID("your_appid")
    v := viper.New()
    v.SetConfigType("prop")
    err := v.AddRemoteProvider("apollo", "your_apollo_endpoint", "your_apollo_namespace")
    //error handle...
    err = v.ReadRemoteConfig()
    //error handle...

    app.WatchRemoteConfigOnChannel() // 启动一个 goroutine 来同步配置更改

    for {
	time.Sleep(1 * time.Second)
	fmt.Println("app.AllSettings:", app.AllSettings())
     }
```

## License

The project is licensed under the [Apache 2 license](https://github.com/taptap/go-apollo/blob/master/LICENSE).

