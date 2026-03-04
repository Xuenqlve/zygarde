package config

import (
	"flag"
	"fmt"
	"os"
)

const (
	DefaultConfigType Type = "file"
	NacosConfigType   Type = "nacos"
	ETCDConfigType    Type = "etcd"
	ConfigTypeKey          = "config-type"
	ConfigFilePathKey      = "config-path"
)

func NewConfig() (Config, error) {
	var configFile = ""
	flagSet := flag.NewFlagSet("app", flag.ContinueOnError)
	flagSet.StringVar(&configFile, "config", "", "config file")
	if err := flagSet.Parse(os.Args[1:]); err != nil {
		return nil, err
	}
	fmt.Printf("config: %s\n", configFile)
	if configFile == "" {
		return nil, fmt.Errorf("缺少参数 -config")
	}
	cfgData, err := utils.ConfigFromFile(configFile)
	if err != nil {
		return nil, errors.Trace(err)
	}
	var configType Type
	if cfgType, ok := cfgData[ConfigTypeKey]; ok {
		configType = cfgType.(Type)
	} else {
		configType = DefaultConfigType
		cfgData[ConfigFilePathKey] = configFile
	}

	config, err := GetConfig(configType)
	if err != nil {
		return nil, errors.Trace(err)
	}
	if err = config.Configure(cfgData); err != nil {
		return nil, errors.Trace(err)
	}
	return config, nil
}
