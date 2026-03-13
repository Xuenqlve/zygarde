package template

import (
	"fmt"
	"sync"

	"github.com/xuenqlve/common/errors"
	"github.com/xuenqlve/zygarde/internal/model"
	"github.com/xuenqlve/zygarde/internal/runtime"
)

func NewMiddlewareKey(middleware, template string) MiddlewareKey {
	return MiddlewareKey{
		middleware: middleware,
		template:   template,
	}
}

type MiddlewareKey struct {
	middleware string
	template   string
}

func (m MiddlewareKey) Key() string {
	return m.middleware + "_" + m.template
}

func (m MiddlewareKey) Middleware() string {
	return m.middleware
}

func (m MiddlewareKey) Template() string {
	return m.template
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
	BuildRuntimeContext(service model.BlueprintService, runtimeType runtime.EnvironmentType) (runtime.EnvironmentContext, error)
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
	if key.Middleware() == "" || key.Template() == "" {
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
		if _, ok := _default_registry[key.Middleware()]; ok {
			return errors.Errorf("default middleware already registered: %s", key.Middleware())
		}
		_default_registry[key.Middleware()] = m
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

func GetDefaultMiddleware(middleware string) (Middleware, error) {
	_middleware_mutex.Lock()
	defer _middleware_mutex.Unlock()
	if _default_registry == nil {
		return nil, errors.New("no default middleware registry exists")
	}
	m, ok := _default_registry[middleware]
	if !ok {
		return nil, errors.Errorf("no default middleware:%s", middleware)
	}
	return m, nil
}

func ResolveMiddleware(input ServiceInput) (Middleware, error) {
	if input.Middleware == "" {
		return nil, errors.New("middleware is required")
	}
	if input.Template == "" {
		return GetDefaultMiddleware(input.Middleware)
	}
	return GetMiddleware(NewMiddlewareKey(input.Middleware, input.Template))
}

func NormalizeServices(inputs []ServiceInput) ([]model.BlueprintService, error) {
	services := make([]model.BlueprintService, 0, len(inputs))
	seenNames := make(map[string]struct{}, len(inputs))

	for i, input := range inputs {
		middleware, err := ResolveMiddleware(input)
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
