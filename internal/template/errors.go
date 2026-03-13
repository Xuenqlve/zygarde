package template

import "errors"

var (
	// ErrMiddlewareRequired indicates that a service input omitted middleware.
	ErrMiddlewareRequired = errors.New("middleware is required")
	// ErrSpecNotFound indicates that no matching middleware/template implementation exists.
	ErrSpecNotFound = errors.New("service spec not found")
	// ErrDefaultSpecNotFound indicates that a middleware has no default template implementation.
	ErrDefaultSpecNotFound = errors.New("default service spec not found")
	// ErrSpecAlreadyRegistered indicates that one middleware/template implementation was registered twice.
	ErrSpecAlreadyRegistered = errors.New("service spec already registered")
	// ErrDefaultSpecAlreadyRegistered indicates that one middleware registered more than one default spec.
	ErrDefaultSpecAlreadyRegistered = errors.New("default service spec already registered")
	// ErrServiceNameRequired indicates that one normalized service still has no name.
	ErrServiceNameRequired = errors.New("service name is required")
	// ErrServiceTemplateRequired indicates that one normalized service still has no template.
	ErrServiceTemplateRequired = errors.New("service template is required")
	// ErrDuplicateServiceName indicates that two normalized services resolved to the same name.
	ErrDuplicateServiceName = errors.New("duplicate service name")
)
