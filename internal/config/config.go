package config

type Database struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	Instance string `yaml:"instance"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type TLS struct {
	PrivateKeyPath  string `yaml:"privateKeyPath"`
	CertificatePath string `yaml:"certificatePath"`
}

type Network struct {
	EntrypointPort   string `yaml:"entrypointPort"`
	TLS              TLS    `yaml:"tls"`
	ExternalHostname string `yaml:"externalHostname"`
}
type Log struct {
	Level    string      `yaml:"level"`
	Location string      `yaml:"location"`
	Rotation LogRotation `yaml:"rotation"`
}

type LogRotation struct {
	MaxSizeMB string `yaml:"maxSizeMB"`
	Count     string `yaml:"count"`
}

type SingleNodeConfiguration struct {
	Database Database `yaml:"database"`
	Network  Network  `yaml:"network"`
	Log      Log      `yaml:"log"`
}
