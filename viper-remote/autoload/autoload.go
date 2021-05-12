package autoload

import (
	"fmt"
	"github.com/spf13/viper"
	"github.com/taptap/go-apollo"
	remote "github.com/taptap/go-apollo/viper-remote"
	"os"
)

const (
	EnvAppID     = "APOLLO_APP_ID"
	EnvSecret    = "APOLLO_ACCESS_KEY"
	EnvMeta      = "APOLLO_META"
	EnvCluster   = "APOLLO_CLUSTER"
	EnvNamespace = "APOLLO_NAMESPACE"
)

const (
	defaultMeta      = "http://apollo.meta"
	defaultCluster   = "default"
	defaultNamespace = "application"
)

func init() {
	appId := getAppID()
	if appId == "" {
		panic(fmt.Sprintf("autoload apollo configs: get app_id failed, please set %s Environment variable", EnvAppID))
	}

	if err := readConfig(appId); err != nil {
		panic(fmt.Errorf("autoload apollo configs: read configs for app_id %s failed, %w", appId, err))
	}
}

func readConfig(appId string) error {
	remote.SetAppID(appId)
	remote.SetApolloOptions(
		apollo.WithClientOptions(
			apollo.WithAccessKey(getSecret()),
		),
		apollo.Cluster(getCluster()),
		apollo.PreloadNamespaces(getNamespace()),
	)

	viper.SetConfigType("prop")
	err := viper.AddRemoteProvider("apollo", getMeta(), getNamespace())
	if err != nil {
		return err
	}
	err = viper.ReadRemoteConfig()
	if err != nil {
		return err
	}

	return nil
}

func getAppID() string {
	return os.Getenv(EnvAppID)
}

func getSecret() string {
	return os.Getenv(EnvSecret)
}

func getMeta() string {
	m := os.Getenv(EnvMeta)
	if m == "" {
		m = defaultMeta
	}
	return m
}

func getCluster() string {
	c := os.Getenv(EnvCluster)
	if c == "" {
		c = defaultCluster
	}

	return c
}

func getNamespace() string {
	c := os.Getenv(EnvNamespace)
	if c == "" {
		c = defaultNamespace
	}

	return c
}
