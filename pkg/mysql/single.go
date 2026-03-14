package mysql

import (
	"fmt"
	"strconv"

	"github.com/xuenqlve/zygarde/internal/model"
	"github.com/xuenqlve/zygarde/internal/runtime"
	tpl "github.com/xuenqlve/zygarde/internal/template"
)

const (
	middlewareName = "mysql"
	singleTemplate = "single"
	defaultPort    = 3306
)

func init() {
	if err := Register(runtime.EnvironmentTypeCompose); err != nil {
		panic(err)
	}
}

// Register registers MySQL specs into the provided registry.
func Register(envType runtime.EnvironmentType) error {
	return tpl.RegisterMiddleware(tpl.NewMiddlewareRuntimeKey(middlewareName, singleTemplate, envType), NewSingleSpec())
}

// NewSingleSpec returns the default MySQL single-node middleware spec.
func NewSingleSpec() tpl.Middleware {
	return &singleSpec{}
}

type singleSpec struct {
	services []model.BlueprintService
}

func (*singleSpec) Middleware() string {
	return middlewareName
}

func (*singleSpec) Template() string {
	return singleTemplate
}

func (*singleSpec) IsDefault() bool {
	return true
}

func (s *singleSpec) Normalize(input tpl.ServiceInput, index int) (model.BlueprintService, error) {
	name := input.Name
	if name == "" {
		name = tpl.DefaultServiceName(s.Middleware(), index)
	}

	values := tpl.MergeValues(s.DefaultValues(), input.Values)
	values["service_name"] = defaultStringValue(values["service_name"], name)
	values["container_name"] = defaultStringValue(values["container_name"], name)

	port, err := normalizePort(values["port"])
	if err != nil {
		return model.BlueprintService{}, fmt.Errorf("normalize mysql single port: %w", err)
	}
	values["port"] = port

	rootPassword, ok := values["root_password"].(string)
	if !ok || rootPassword == "" {
		return model.BlueprintService{}, fmt.Errorf("normalize mysql single root_password: must be a non-empty string")
	}

	return model.BlueprintService{
		Name:       name,
		Middleware: s.Middleware(),
		Template:   s.Template(),
		Values:     values,
	}, nil
}

func (s *singleSpec) Configure(input tpl.ServiceInput, index int) (model.BlueprintService, error) {
	service, err := s.Normalize(input, index)
	if err != nil {
		return model.BlueprintService{}, err
	}
	if err := s.Validate(service); err != nil {
		return model.BlueprintService{}, err
	}
	s.services = append(s.services, service)
	return service, nil
}

func (s *singleSpec) BuildRuntimeContexts(runtimeType runtime.EnvironmentType) ([]runtime.EnvironmentContext, error) {
	contexts := make([]runtime.EnvironmentContext, 0, len(s.services))
	for _, service := range s.services {
		if err := s.Validate(service); err != nil {
			return nil, err
		}

		contexts = append(contexts, runtime.EnvironmentContext{
			RuntimeType: runtimeType,
			ServiceName: service.Name,
			Middleware:  service.Middleware,
			Template:    service.Template,
			Values:      tpl.MergeValues(nil, service.Values),
		})
	}
	return contexts, nil
}

func (*singleSpec) Validate(service model.BlueprintService) error {
	if service.Name == "" {
		return tpl.ErrServiceNameRequired
	}
	if service.Template == "" {
		return tpl.ErrServiceTemplateRequired
	}
	if service.Middleware != middlewareName {
		return fmt.Errorf("mysql single validate: unexpected middleware %q", service.Middleware)
	}
	if service.Template != singleTemplate {
		return fmt.Errorf("mysql single validate: unexpected template %q", service.Template)
	}

	port, err := normalizePort(service.Values["port"])
	if err != nil {
		return fmt.Errorf("mysql single validate port: %w", err)
	}
	if port <= 0 {
		return fmt.Errorf("mysql single validate port: must be greater than 0")
	}

	rootPassword, ok := service.Values["root_password"].(string)
	if !ok || rootPassword == "" {
		return fmt.Errorf("mysql single validate root_password: must be a non-empty string")
	}

	serviceName, ok := service.Values["service_name"].(string)
	if !ok || serviceName == "" {
		return fmt.Errorf("mysql single validate service_name: must be a non-empty string")
	}

	containerName, ok := service.Values["container_name"].(string)
	if !ok || containerName == "" {
		return fmt.Errorf("mysql single validate container_name: must be a non-empty string")
	}

	return nil
}

func defaultStringValue(value any, fallback string) string {
	current, ok := value.(string)
	if !ok || current == "" {
		return fallback
	}
	return current
}

func (*singleSpec) DefaultValues() map[string]any {
	return map[string]any{
		"port":          defaultPort,
		"root_password": "root",
	}
}

func normalizePort(value any) (int, error) {
	switch port := value.(type) {
	case int:
		return port, nil
	case int8:
		return int(port), nil
	case int16:
		return int(port), nil
	case int32:
		return int(port), nil
	case int64:
		return int(port), nil
	case uint:
		return int(port), nil
	case uint8:
		return int(port), nil
	case uint16:
		return int(port), nil
	case uint32:
		return int(port), nil
	case uint64:
		return int(port), nil
	case float32:
		return int(port), nil
	case float64:
		return int(port), nil
	case string:
		parsed, err := strconv.Atoi(port)
		if err != nil {
			return 0, err
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("unsupported port type %T", value)
	}
}
