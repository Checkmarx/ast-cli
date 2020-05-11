package config

type AIOConfiguration struct {
	Database struct {
		Host     interface{} `yaml:"host"`
		Port     interface{} `yaml:"port"`
		Name     interface{} `yaml:"name"`
		Username interface{} `yaml:"username"`
		Password interface{} `yaml:"password"`
	} `yaml:"database"`
	Network struct {
		EntrypointPort    interface{} `yaml:"entrypointPort"`
		EntrypointTLSPort interface{} `yaml:"entrypointTLSPort"`
		PrivateKeyFile    interface{} `yaml:"privateKeyFile"`
		PublicKeyFile     interface{} `yaml:"publicKeyFile"`
	} `yaml:"network"`
	ObjectStore struct {
		AccessKeyID     interface{} `yaml:"accessKeyId"`
		SecretAccessKey interface{} `yaml:"secretAccessKey"`
	} `yaml:"objectStore"`
	MessageQueue struct {
		Username interface{} `yaml:"username"`
		Password interface{} `yaml:"password"`
	} `yaml:"messageQueue"`
	AccessControl struct {
		Username interface{} `yaml:"username"`
		Password interface{} `yaml:"password"`
	} `yaml:"accessControl"`
	Log struct {
		Level    interface{} `yaml:"level"`
		Rotation struct {
			MaxSizeMB  interface{} `yaml:"maxSizeMB"`
			MaxAgeDays interface{} `yaml:"maxAgeDays"`
		} `yaml:"rotation"`
	} `yaml:"log"`
}
