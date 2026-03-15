package mock

import (
	"fmt"

	"github.com/xuenqlve/zygarde/internal/model"
	"github.com/xuenqlve/zygarde/internal/runtime"
	tpl "github.com/xuenqlve/zygarde/internal/template"
)

const (
	middlewareName = "mock"
	echoTemplate   = "echo"
)

func init() {
	if err := Register(runtime.EnvironmentTypeCompose); err != nil {
		panic(err)
	}
}

// Register registers the mock echo middleware for one runtime.
func Register(envType runtime.EnvironmentType) error {
	return tpl.RegisterMiddleware(
		tpl.NewMiddlewareRuntimeKey(middlewareName, echoTemplate, envType),
		NewEchoSpec(),
	)
}

// NewEchoSpec returns the mock echo middleware implementation.
func NewEchoSpec() tpl.Middleware {
	return &echoSpec{}
}

type echoSpec struct {
	services []model.BlueprintService
}

func (*echoSpec) Middleware() string {
	return middlewareName
}

func (*echoSpec) Template() string {
	return echoTemplate
}

func (*echoSpec) IsDefault() bool {
	return true
}

func (s *echoSpec) Configure(input tpl.ServiceInput, index int) (model.BlueprintService, error) {
	name := input.Name
	if name == "" {
		name = tpl.DefaultServiceName(s.Middleware(), index)
	}

	values := tpl.MergeValues(nil, input.Values)
	if values == nil {
		values = map[string]any{}
	}

	service := model.BlueprintService{
		Name:       name,
		Middleware: s.Middleware(),
		Template:   s.Template(),
		Values:     values,
	}

	fmt.Printf(
		"[mock/echo] configure index=%d name=%s middleware=%s template=%s values=%v\n",
		index,
		service.Name,
		service.Middleware,
		service.Template,
		service.Values,
	)

	s.services = append(s.services, service)
	return service, nil
}

func (s *echoSpec) BuildRuntimeContexts(runtimeType runtime.EnvironmentType) ([]runtime.EnvironmentContext, error) {
	contexts := make([]runtime.EnvironmentContext, 0, len(s.services))
	for _, service := range s.services {
		contexts = append(contexts, runtime.ComposeContext{
			EnvType:     runtimeType,
			ServiceName: service.Name,
			Middleware:  service.Middleware,
			Template:    service.Template,
			Service: runtime.ServiceSpec{
				Image:         "alpine:3.20",
				ContainerName: service.Name,
				Command:       []string{"sleep", "infinity"},
			},
			Assets: []runtime.AssetSpec{
				{
					Name:      "mock-build",
					PathKey:   "build_script",
					Content:   "echo \"Mock compose stack started\"\n",
					Mode:      0o755,
					MergeMode: runtime.AssetMergeScript,
				},
				{
					Name:      "mock-check",
					PathKey:   "check_script",
					Content:   "docker compose ps\n",
					Mode:      0o755,
					MergeMode: runtime.AssetMergeScript,
				},
				{
					Name:      "mock-readme",
					PathKey:   "readme_file",
					Content:   fmt.Sprintf("# Mock %s\n\n- middleware: %s\n", service.Name, service.Middleware),
					Mode:      0o644,
					MergeMode: runtime.AssetMergeReadme,
				},
			},
			Metadata: tpl.MergeValues(nil, service.Values),
		})
	}
	return contexts, nil
}
