package model

// RuntimeLayout describes the filesystem layout used by one environment.
type RuntimeLayout struct {
	RootDir      string
	ComposeFile  string
	MetadataFile string
	LogsDir      string
	DataDir      string
}
