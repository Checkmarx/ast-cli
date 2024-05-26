package scarealtime

var GetPackageManagerFromResolvingModuleType = map[string]string{
	"composer":  "Php",
	"gomodules": "Go",
	"pip":       "Python",
	"poetry":    "Python",
	"rubygems":  "Ruby",
	"npm":       "Npm",
	"yarn":      "Npm",
	"bower":     "Npm",
	"lerna":     "Npm",
	"sbt":       "Maven",
	"ivy":       "Maven",
	"maven":     "Maven",
	"gradle":    "Maven",
	"swiftpm":   "Ios",
	"carthage":  "Ios",
	"cocoapods": "Ios",
	"nuget":     "Nuget",
	"cpp":       "Cpp",
}
