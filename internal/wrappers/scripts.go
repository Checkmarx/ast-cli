package wrappers

type ScriptsWrapper interface {
	GetDotEnvFilePath() string
	GetInstallScriptPath() string
	GetUpScriptPath() string
	GetDownScriptPath() string
}
