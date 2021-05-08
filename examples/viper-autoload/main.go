package main

import (
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
	_ "github.com/taptap/go-apollo/viper-remote/autoload"
	"log"
)

// 在启动程序前设置你的系统环境变量
// os.Setenv("APOLLO_APP_ID", "TapTap")
// os.Setenv("APOLLO_ACCESS_KEY", "Your Access Key")
// os.Setenv("APOLLO_META", "http://apollo.meta")
// os.Setenv("APOLLO_CLUSTER", "Your Cluster")
// os.Setenv("APOLLO_NAMESPACE", "application")

type Config struct {
	Name      string
	SnakeCase struct {
		Foo string
		Bar string
	}
	CamelCase struct {
		Foo string
	} `json:"camel_case"`
}

func main() {
	// Apollo key-value 配置
	//
	// name=Nobody
	// SnakeCase.Foo=SnakeFoo
	// SnakeCase.Bar=SnakeBar
	// camel_case.foo=CamelFoo
	log.Println(viper.AllSettings())

	// 获取单条配置
	viper.GetString("name") // = Nobody
	// viper key 大小写不敏感，也可以使用 snakeCase.foo
	viper.GetString("snakeCase.Foo")  // = SnakeFoo
	viper.GetString("camel_case.foo") // = CamelFoo

	myConfig := &Config{}
	// 解析
	_ = viper.Unmarshal(myConfig, func(config *mapstructure.DecoderConfig) {
		config.TagName = "json"
	})
	log.Println(myConfig)
}
