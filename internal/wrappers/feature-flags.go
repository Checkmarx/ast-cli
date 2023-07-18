package wrappers

var FeatureFlagsBaseMap = CommandFlags{
	{
		CommandName: "scan create",
		FeatureFlags: []FlagBase{
			{
				Name:    "PACKAGE_ENFORCEMENT_ENABLED",
				Default: true,
			},
		},
	},
}

type FeatureFlagsWrapper interface {
	GetAll() (*FeatureFlagsResponseModel, error)
}

type FeatureFlagsResponseModel []struct {
	Name   string `json:"name"`
	Status bool   `json:"status"`
}

type Flag struct {
	Status  bool
	Payload interface{}
}

type CommandFlags []struct {
	CommandName  string
	FeatureFlags []FlagBase
}

type FlagBase struct {
	Name    string
	Default bool
}
