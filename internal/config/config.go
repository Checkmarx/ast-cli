package config

type Execution struct {
	Type string `yaml:"type"`
}

type Database struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	Name     string `yaml:"name"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}
type TLS struct {
	PrivateKeyPath  string `yaml:"privateKeyPath"`
	CertificatePath string `yaml:"certificatePath"`
}

type Network struct {
	EntrypointPort           string `yaml:"entrypointPort"`
	EntrypointTLSPort        string `yaml:"entrypointTLSPort"`
	FullyQualifiedDomainName string `yaml:"fqdn"`
	TLS                      TLS    `yaml:"tls"`
	ExternalAccessIP         string `yaml:"externalAccessIP"`
}

type ObjectStore struct {
	AccessKeyID     string `yaml:"accessKeyId"`
	SecretAccessKey string `yaml:"secretAccessKey"`
}

type MessageQueue struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type AccessControl struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type Log struct {
	Level    string      `yaml:"level"`
	Rotation LogRotation `yaml:"rotation"`
}

type LogRotation struct {
	MaxSizeMB  string `yaml:"maxSizeMB"`
	MaxAgeDays string `yaml:"maxAgeDays"`
}

type SingleNodeConfiguration struct {
	Execution     Execution     `yaml:"execution"`
	Database      Database      `yaml:"database"`
	Network       Network       `yaml:"network"`
	ObjectStore   ObjectStore   `yaml:"objectStore"`
	MessageQueue  MessageQueue  `yaml:"messageQueue"`
	AccessControl AccessControl `yaml:"accessControl"`
	Log           Log           `yaml:"log"`
}
