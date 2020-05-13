package wrappers

import "path"

func NewScriptsFolderWrapper(dotEnv, dir, install, up, down string) ScriptsWrapper {
	return &ScriptsFolderWrapper{
		dotEnvFilePath:  dotEnv,
		scriptsDir:      dir,
		installFilename: install,
		upFilename:      up,
		downFilename:    down,
	}
}

type ScriptsFolderWrapper struct {
	dotEnvFilePath  string
	scriptsDir      string
	installFilename string
	upFilename      string
	downFilename    string
}

func (s *ScriptsFolderWrapper) GetDotEnvFilePath() string {
	return s.dotEnvFilePath
}

func (s *ScriptsFolderWrapper) GetInstallScriptPath() string {
	return path.Join(s.scriptsDir, s.installFilename)
}

func (s *ScriptsFolderWrapper) GetUpScriptPath() string {
	return path.Join(s.scriptsDir, s.upFilename)
}

func (s *ScriptsFolderWrapper) GetDownScriptPath() string {
	return path.Join(s.scriptsDir, s.downFilename)
}
