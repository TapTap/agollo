package autoload

import (
	"fmt"
	"github.com/spf13/viper"
	"github.com/taptap/go-apollo"
	remote "github.com/taptap/go-apollo/viper-remote"
	"os"
)

const (
	envAppID     = "APOLLO_APP_ID"
	envSecret    = "APOLLO_ACCESS_KEY"
	envMeta      = "APOLLO_META"
	envCluster   = "APOLLO_CLUSTER"
	envNamespace = "APOLLO_NAMESPACE"
)

const (
	defaultMeta      = "http://apollo.meta"
	defaultCluster   = "default"
	defaultNamespace = "application"
)

func init() {
	appId := getAppID()
	if appId == "" {
		panic(fmt.Sprintf("autoload apollo configs: get app_id failed, please set %s environment variable", envAppID))
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
	return os.Getenv(envAppID)
}

func getSecret() string {
	return os.Getenv(envSecret)
}

func getMeta() string {
	m := os.Getenv(envMeta)
	if m == "" {
		m = defaultMeta
	}
	return m
}

func getCluster() string {
	c := os.Getenv(envCluster)
	if c == "" {
		c = defaultCluster
	}

	return c
}

func getNamespace() string {
	c := os.Getenv(envNamespace)
	if c == "" {
		c = defaultNamespace
	}

	return c
}
