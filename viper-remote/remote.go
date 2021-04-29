package remote

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/spf13/viper"
	"github.com/taptap/go-apollo"
)

var (
	ErrUnsupportedProvider = errors.New("This configuration manager is not supported")

	_ viperConfigManager = apolloConfigManager{}
	// getConfigManager方法每次返回新对象导致缓存无效，
	// 这里通过endpoint作为key复一个对象
	// key: endpoint+appid value: apollo.Apollo
	apolloMap sync.Map
)

var (
	// apollod的appid
	appID string
	// 默认为properties，apollo默认配置文件格式
	defaultConfigType = "properties"
	// 默认创建Apollo的Option
	defaultApolloOptions = []apollo.Option{
		apollo.AutoFetchOnCacheMiss(),
		apollo.FailTolerantOnBackupExists(),
	}
)

func SetAppID(appid string) {
	appID = appid
}

func SetConfigType(ct string) {
	defaultConfigType = ct
}

func SetApolloOptions(opts ...apollo.Option) {
	defaultApolloOptions = opts
}

type viperConfigManager interface {
	Get(key string) ([]byte, error)
	Watch(key string, stop chan bool) <-chan *viper.RemoteResponse
}

type apolloConfigManager struct {
	apollo apollo.Apollo
}

func newApolloConfigManager(appid, endpoint string, opts []apollo.Option) (*apolloConfigManager, error) {
	if appid == "" {
		return nil, errors.New("The appid is not set")
	}

	ag, err := newApollo(appid, endpoint, opts)
	if err != nil {
		return nil, err
	}

	return &apolloConfigManager{
		apollo: ag,
	}, nil

}

func newApollo(appid, endpoint string, opts []apollo.Option) (apollo.Apollo, error) {
	i, found := apolloMap.Load(endpoint + "/" + appid)
	if !found {
		ag, err := apollo.New(
			endpoint,
			appid,
			opts...,
		)
		if err != nil {
			return nil, err
		}

		// 监听并同步apollo配置
		ag.Start()

		apolloMap.Store(endpoint + "/" + appid, ag)

		return ag, nil
	}
	return i.(apollo.Apollo), nil
}

func (cm apolloConfigManager) Get(namespace string) ([]byte, error) {
	configs := cm.apollo.GetNameSpace(namespace)
	return marshalConfigs(getConfigType(namespace), configs)
}

func marshalConfigs(configType string, configs map[string]interface{}) ([]byte, error) {
	var bts []byte
	var err error
	switch configType {
	case "json", "yml", "yaml", "xml":
		content := configs["content"]
		if content != nil {
			bts = []byte(content.(string))
		}
	case "properties":
		bts, err = marshalProperties(configs)
	}
	return bts, err
}

func (cm apolloConfigManager) Watch(namespace string, stop chan bool) <-chan *viper.RemoteResponse {
	resp := make(chan *viper.RemoteResponse)
	backendResp := cm.apollo.WatchNamespace(namespace, stop)
	go func() {
		for {
			select {
			case <-stop:
				return
			case r := <-backendResp:
				if r.Error != nil {
					resp <- &viper.RemoteResponse{
						Value: nil,
						Error: r.Error,
					}
					continue
				}

				configType := getConfigType(namespace)
				value, err := marshalConfigs(configType, r.NewValue)

				resp <- &viper.RemoteResponse{Value: value, Error: err}
			}
		}
	}()
	return resp
}

type configProvider struct {
}

func (rc configProvider) Get(rp viper.RemoteProvider) (io.Reader, error) {
	cmt, err := getConfigManager(rp)
	if err != nil {
		return nil, err
	}

	var b []byte
	switch cm := cmt.(type) {
	case viperConfigManager:
		b, err = cm.Get(rp.Path())
	}

	if err != nil {
		return nil, err
	}
	return bytes.NewReader(b), nil
}

func (rc configProvider) Watch(rp viper.RemoteProvider) (io.Reader, error) {
	cmt, err := getConfigManager(rp)
	if err != nil {
		return nil, err
	}

	var resp []byte
	switch cm := cmt.(type) {
	case viperConfigManager:
		resp, err = cm.Get(rp.Path())
	}

	if err != nil {
		return nil, err
	}

	return bytes.NewReader(resp), nil
}

func (rc configProvider) WatchChannel(rp viper.RemoteProvider) (<-chan *viper.RemoteResponse, chan bool) {
	cmt, err := getConfigManager(rp)
	if err != nil {
		return nil, nil
	}

	switch cm := cmt.(type) {
	case viperConfigManager:
		quitwc := make(chan bool)
		viperResponsCh := cm.Watch(rp.Path(), quitwc)
		return viperResponsCh, quitwc
	default:
		return nil, nil
	}
}

func getConfigManager(rp viper.RemoteProvider) (interface{}, error) {
	if rp.SecretKeyring() != "" {
		kr, err := os.Open(rp.SecretKeyring())
		if err != nil {
			return nil, err
		}
		defer kr.Close()

		switch rp.Provider() {
		case "apollo":
			return nil, errors.New("The Apollo configuration manager is not encrypted")
		default:
			return nil, ErrUnsupportedProvider
		}
	} else {
		switch rp.Provider() {
		case "apollo":
			return newApolloConfigManager(appID, rp.Endpoint(), defaultApolloOptions)
		default:
			return nil, ErrUnsupportedProvider
		}
	}
}

// 配置文件有多种格式，例如：properties、xml、yml、yaml、json等。同样Namespace也具有这些格式。在Portal UI中可以看到“application”的Namespace上有一个“properties”标签，表明“application”是properties格式的。
// 如果使用Http接口直接调用时，对应的namespace参数需要传入namespace的名字加上后缀名，如datasources.json。
func getConfigType(namespace string) string {
	ext := filepath.Ext(namespace)

	if len(ext) > 1 {
		fileExt := ext[1:]
		// 还是要判断一下碰到，TEST.Namespace1
		// 会把Namespace1作为文件扩展名
		for _, e := range viper.SupportedExts {
			if e == fileExt {
				return fileExt
			}
		}
	}

	return defaultConfigType
}

func init() {
	viper.SupportedRemoteProviders = append(
		viper.SupportedRemoteProviders,
		"apollo",
	)
	viper.RemoteConfig = &configProvider{}
}
