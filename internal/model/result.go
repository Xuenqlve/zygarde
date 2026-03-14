package model

// RenderResult describes the output of the rendering stage.
type RenderResult struct {
	Content        string
	PrimaryFile    string
	ComposeVersion string
	Services       []string
	Warnings       []string
}

// DeploymentResult describes the result of one deployment action.
type DeploymentResult struct {
	Success      bool
	ProjectName  string
	Output       string
	ErrorMessage string
}
