package config

import (
	"fmt"
	"reflect"
	"sync"
)

type AppConfig struct {
}

type Config interface {
	Configure(data map[string]any) error
	Parse() (AppConfig, error)
	Reload() error
	Close() error
}

type (
	Factory func() Config
	Type    string
)

var (
	_config_registry map[Type]Factory
	_config_mutext   sync.Mutex
)

func RegisterConfigFactory(configType Type, configFactory Factory) {
	_config_mutext.Lock()
	defer _config_mutext.Unlock()
	if _config_registry == nil {
		_config_registry = make(map[Type]Factory)
	}
	_, ok := _config_registry[configType]
	if ok {
		panic(fmt.Sprintf("config factory already exists, type:%v ", configType))
	}

	_config_registry[configType] = configFactory
}

func RegisterConfig(configType Type, config Config, singleton bool) {
	var configFactory Factory
	if singleton {
		configFactory = func() Config {
			return config
		}
	} else {
		configFactory = func() Config {
			return reflect.New(reflect.TypeOf(config).Elem()).Interface().(Config)
		}
	}
	RegisterConfigFactory(configType, configFactory)
}

func GetConfig(configType Type) (Config, error) {
	_config_mutext.Lock()
	defer _config_mutext.Unlock()

	factory, ok := _config_registry[configType]
	if !ok {
		return nil, fmt.Errorf("config empty plugin name: %v", configType)
	}
	return factory(), nil
}
