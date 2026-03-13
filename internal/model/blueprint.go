package model

// Blueprint describes a deployable combination of template references.
type Blueprint struct {
	Name        string
	Version     string
	Description string
	Templates   []BlueprintTemplateRef
	Metadata    map[string]string
}

// BlueprintTemplateRef binds a template and the values used for rendering it.
type BlueprintTemplateRef struct {
	TemplateName    string
	TemplateVersion string
	Alias           string
	Values          map[string]string
}
