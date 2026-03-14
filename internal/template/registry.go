package template

import (
	"fmt"
	"sync"

	"github.com/xuenqlve/common/errors"
	"github.com/xuenqlve/zygarde/internal/model"
	"github.com/xuenqlve/zygarde/internal/runtime"
)

func NewMiddlewareKey(middleware, template string) MiddlewareKey {
	return NewMiddlewareRuntimeKey(middleware, template, "")
}

type MiddlewareKey struct {
	middleware string
	template   string
	runtime    runtime.EnvironmentType
}

func NewMiddlewareRuntimeKey(middleware, template string, envType runtime.EnvironmentType) MiddlewareKey {
	return MiddlewareKey{
		middleware: middleware,
		template:   template,
		runtime:    envType,
	}
}

func (m MiddlewareKey) Key() string {
	return m.middleware + "_" + m.template + "_" + string(m.runtime)
}

func (m MiddlewareKey) Middleware() string {
	return m.middleware
}

func (m MiddlewareKey) Template() string {
	return m.template
}

func (m MiddlewareKey) EnvironmentType() runtime.EnvironmentType {
	return m.runtime
}

var (
	_middleware_registry map[MiddlewareKey]Middleware
	_default_registry    map[string]Middleware
	_middleware_mutex    sync.Mutex
)

type Middleware interface {
	Middleware() string
	Template() string
	IsDefault() bool
	Configure(input ServiceInput, index int) (model.BlueprintService, error)
	BuildRuntimeContexts(runtimeType runtime.EnvironmentType) ([]runtime.EnvironmentContext, error)
}

type ServiceInput struct {
	Name       string
	Middleware string
	Template   string
	Values     map[string]any
}

func RegisterMiddleware(key MiddlewareKey, m Middleware) error {
	_middleware_mutex.Lock()
	defer _middleware_mutex.Unlock()

	if _middleware_registry == nil {
		_middleware_registry = make(map[MiddlewareKey]Middleware)
	}
	if _default_registry == nil {
		_default_registry = make(map[string]Middleware)
	}
	if key.Middleware() == "" || key.Template() == "" || key.EnvironmentType() == "" {
		return errors.New("middleware key requires middleware and template")
	}
	if key.Middleware() != m.Middleware() || key.Template() != m.Template() {
		return errors.Errorf("middleware key mismatch: key=%s/%s middleware=%s/%s", key.Middleware(), key.Template(), m.Middleware(), m.Template())
	}

	_, ok := _middleware_registry[key]
	if ok {
		return errors.Errorf("middleware already registered: %s", key.Key())
	}

	if m.IsDefault() {
		defaultKey := defaultRegistryKey(key.Middleware(), key.EnvironmentType())
		if _, ok := _default_registry[defaultKey]; ok {
			return errors.Errorf("default middleware already registered: %s", defaultKey)
		}
		_default_registry[defaultKey] = m
	}

	_middleware_registry[key] = m
	return nil
}

func GetMiddlewareSet() (map[MiddlewareKey]Middleware, error) {
	_middleware_mutex.Lock()
	defer _middleware_mutex.Unlock()
	if _middleware_registry == nil {
		return nil, errors.New("no middleware registry exists")
	}

	out := make(map[MiddlewareKey]Middleware, len(_middleware_registry))
	for key, middleware := range _middleware_registry {
		out[key] = middleware
	}
	return out, nil
}

func GetMiddleware(key MiddlewareKey) (Middleware, error) {
	_middleware_mutex.Lock()
	defer _middleware_mutex.Unlock()
	if _middleware_registry == nil {
		return nil, errors.New("no middleware registry exists")
	}
	middleware, ok := _middleware_registry[key]
	if !ok {
		return nil, errors.Errorf("no middleware key:%s", key.Key())
	}
	return middleware, nil
}

func GetDefaultMiddleware(middleware string, envType runtime.EnvironmentType) (Middleware, error) {
	_middleware_mutex.Lock()
	defer _middleware_mutex.Unlock()
	if _default_registry == nil {
		return nil, errors.New("no default middleware registry exists")
	}
	m, ok := _default_registry[defaultRegistryKey(middleware, envType)]
	if !ok {
		return nil, errors.Errorf("no default middleware:%s envType:%s", middleware, envType)
	}
	return m, nil
}

func ResolveMiddleware(input ServiceInput, envType runtime.EnvironmentType) (Middleware, error) {
	if input.Middleware == "" {
		return nil, errors.New("middleware is required")
	}
	if input.Template == "" {
		return GetDefaultMiddleware(input.Middleware, envType)
	}
	return GetMiddleware(NewMiddlewareRuntimeKey(input.Middleware, input.Template, envType))
}

func NormalizeServices(inputs []ServiceInput, envType runtime.EnvironmentType) ([]model.BlueprintService, error) {
	services := make([]model.BlueprintService, 0, len(inputs))
	seenNames := make(map[string]struct{}, len(inputs))

	for i, input := range inputs {
		middleware, err := ResolveMiddleware(input, envType)
		if err != nil {
			return nil, err
		}

		service, err := middleware.Configure(input, i+1)
		if err != nil {
			return nil, err
		}
		if service.Name == "" {
			return nil, errors.New("service name is required")
		}
		if _, exists := seenNames[service.Name]; exists {
			return nil, errors.Errorf("duplicate service name:%s", service.Name)
		}
		seenNames[service.Name] = struct{}{}
		services = append(services, service)
	}

	return services, nil
}

func defaultRegistryKey(middleware string, envType runtime.EnvironmentType) string {
	return middleware + "_" + string(envType)
}

func DefaultServiceName(middleware string, index int) string {
	return fmt.Sprintf("%s-%d", middleware, index)
}

func MergeValues(defaults, overrides map[string]any) map[string]any {
	merged := make(map[string]any, len(defaults)+len(overrides))
	for key, value := range defaults {
		merged[key] = value
	}
	for key, value := range overrides {
		merged[key] = value
	}
	return merged
}
