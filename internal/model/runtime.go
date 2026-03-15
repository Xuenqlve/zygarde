package model

// RuntimeLayout describes the filesystem layout used by one environment.
type RuntimeLayout struct {
	// RootDir 是当前环境的工作根目录。
	RootDir string
	// RenderDir 是运行时产物输出目录；Compose 当前与 RootDir 相同。
	RenderDir string
	// ComposeFile 是 Compose runtime 生成的 docker-compose.yml 文件路径。
	ComposeFile string
	// EnvFile 是 Compose runtime 生成的 .env 文件路径。
	EnvFile string
	// BuildScript 是 Compose runtime 生成的 build.sh 文件路径。
	BuildScript string
	// CheckScript 是 Compose runtime 生成的 check.sh 文件路径。
	CheckScript string
	// ReadmeFile 是 Compose runtime 生成的 README.md 文件路径。
	ReadmeFile string
	// MetadataFile 是环境元数据持久化文件路径。
	MetadataFile string
	// LogsDir 是当前环境日志目录。
	LogsDir string
	// DataDir 是当前环境运行数据目录。
	DataDir string
}
