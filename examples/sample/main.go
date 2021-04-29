package main

import (
	"fmt"
	"os"
	"time"

	apollo "github.com/taptap/go-apollo"
)

func main() {
	// 通过默认根目录下的app.properties初始化apollo
	err := apollo.InitWithDefaultConfigFile(
		apollo.WithLogger(apollo.NewLogger(apollo.LoggerWriter(os.Stdout))), // 打印日志信息
		apollo.PreloadNamespaces("TEST.Namespace"),                          // 预先加载的namespace列表，如果是通过配置启动，会在app.properties配置的基础上追加
		apollo.AutoFetchOnCacheMiss(),                                       // 在配置未找到时，去apollo的带缓存的获取配置接口，获取配置
		apollo.FailTolerantOnBackupExists(),                                 // 在连接apollo失败时，如果在配置的目录下存在.apollo备份配置，会读取备份在服务器无法连接的情况下

		// apollo 官方签名认证
		// apollo.WithClientOptions(
		// 	apollo.WithAccessKey("your_access_key"),
		// ),

		// 自定义签名认证
		// apollo.WithClientOptions(
		// 	apollo.WithSignatureFunc(
		// 		func(*apollo.SignatureContext) apollo.Header {
		// 			return apollo.Header{
		// 				"Authorization": "basic xxxxxxxx",
		// 			}
		// 		}),
		// ),
	)
	if err != nil {
		panic(err)
	}

	/*
		通过指定配置文件地址的方式初始化
		apollo.InitWithConfigFile(configFilePath string, opts ....Option)

		参数形式初始化apollo的方式，适合二次封装
		apollo.Init(
			"localhost:8080",
			"AppTest",
		        opts...,
		)
	*/

	// 获取默认配置中cluster=default namespace=application key=Name的值
	fmt.Println("timeout:", apollo.Get("timeout"))

	// 获取默认配置中cluster=default namespace=application key=timeout的值，提供默认值返回
	fmt.Println("YourConfigKey:", apollo.Get("YourConfigKey", apollo.WithDefault("YourDefaultValue")))

	// 获取默认配置中cluster=default namespace=Test.Namespace key=timeout的值，提供默认值返回
	fmt.Println("YourConfigKey2:", apollo.Get("YourConfigKey2", apollo.WithDefault("YourDefaultValue"), apollo.WithNamespace("YourNamespace")))

	// 获取namespace下的所有配置项
	fmt.Println("Configuration of the namespace:", apollo.GetNameSpace("application"))

	// TEST.Namespace1是非预加载的namespace
	// apollo初始化是带上apollo.AutoFetchOnCacheMiss()可选项的话
	// 陪到非预加载的namespace，会去apollo缓存接口获取配置
	// 未配置的话会返回空或者传入的默认值选项
	fmt.Println(apollo.Get("timeout", apollo.WithDefault("foo"), apollo.WithNamespace("TEST.Namespace1")))

	// 如果想监听并同步服务器配置变化，启动apollo长轮训
	// 返回一个期间发生错误的error channel,按照需要去处理
	errorCh := apollo.Start()
	defer apollo.Stop()

	// 监听apollo配置更改事件
	// 返回namespace和其变化前后的配置,以及可能出现的error
	watchCh := apollo.Watch()

	stop := make(chan bool)
	watchNamespace := "YourNamespace"
	watchNSCh := apollo.WatchNamespace(watchNamespace, stop)

	appNSCh := apollo.WatchNamespace("application", stop)
	go func() {
		for {
			select {
			case err := <-errorCh:
				fmt.Println("Error:", err)
			case resp := <-watchCh:
				fmt.Println("Watch Apollo:", resp)
			case resp := <-watchNSCh:
				fmt.Println("Watch Namespace", watchNamespace, resp)
			case resp := <-appNSCh:
				fmt.Println("Watch Namespace", "application", resp)
			case <-time.After(time.Second):
				fmt.Println("timeout:", apollo.Get("timeout"))
			}
		}
	}()

	select {}
}
