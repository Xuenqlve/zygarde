package model

// Template describes a reusable deployment template definition.
type Template struct {
	Name        string
	Version     string
	Middleware  string
	Scenario    string
	Description string
	Source      string
	Variables   []TemplateVariable
	Metadata    map[string]string
}

// TemplateVariable describes an input accepted by a template.
type TemplateVariable struct {
	Name         string
	Type         string
	Required     bool
	DefaultValue string
	Description  string
}
