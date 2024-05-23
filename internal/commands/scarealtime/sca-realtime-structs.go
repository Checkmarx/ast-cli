package scarealtime

import "time"

type ScaResultsFile struct {
	ScanMetadata struct {
		StartTime     time.Time `json:"StartTime"`
		ScanPath      string    `json:"ScanPath"`
		ScanArguments struct {
			ProjectDownloadURL interface{} `json:"ProjectDownloadUrl"`
			ScanID             string      `json:"ScanID"`
			TenantID           string      `json:"TenantId"`
			ExcludePatterns    struct {
				Patterns []interface{} `json:"Patterns"`
			} `json:"ExcludePatterns"`
			IgnoreDevDependencies               bool        `json:"IgnoreDevDependencies"`
			ShouldResolveDependenciesLocally    bool        `json:"ShouldResolveDependenciesLocally"`
			NpmPartialResultsFallbackScriptPath interface{} `json:"NpmPartialResultsFallbackScriptPath"`
			EnvironmentVariables                struct {
			} `json:"EnvironmentVariables"`
			ShouldUseHoistFlagWhenUseLerna              bool `json:"ShouldUseHoistFlagWhenUseLerna"`
			ShouldResolvePartialResults                 bool `json:"ShouldResolvePartialResults"`
			ProjectRelativePathsToPythonVersionsMapping struct {
			} `json:"ProjectRelativePathsToPythonVersionsMapping"`
			AdditionalScanArguments struct {
			} `json:"AdditionalScanArguments"`
			ExtractArchives                []string      `json:"ExtractArchives"`
			ExtractDepth                   int           `json:"ExtractDepth"`
			GradleExcludedScopes           []interface{} `json:"GradleExcludedScopes"`
			GradleIncludedScopes           []interface{} `json:"GradleIncludedScopes"`
			GradleDevDependenciesScopes    []interface{} `json:"GradleDevDependenciesScopes"`
			GradleModulesToIgnore          []interface{} `json:"GradleModulesToIgnore"`
			GradleModulesToInclude         []interface{} `json:"GradleModulesToInclude"`
			GradlePluginDependenciesScopes []interface{} `json:"GradlePluginDependenciesScopes"`
			Proxies                        struct {
			} `json:"Proxies"`
			NugetCliPath        string      `json:"NugetCliPath"`
			IvyReportTarget     interface{} `json:"IvyReportTarget"`
			IvyReportFilesDir   interface{} `json:"IvyReportFilesDir"`
			EnableContainerScan bool        `json:"EnableContainerScan"`
		} `json:"ScanArguments"`
		ScanDiagnostics struct {
			ShouldResolveDependenciesLocally                  bool    `json:"ShouldResolveDependenciesLocally"`
			ScopeMilliseconds                                 int     `json:"scopeMilliseconds"`
			ResolveDependenciesForFilePomXMLScopeMilliseconds int     `json:"ResolveDependenciesForFile[pom.xml].scopeMilliseconds"`
			ShouldResolvePartialResults                       string  `json:"ShouldResolvePartialResults"`
			EnvironmentVariables                              string  `json:"EnvironmentVariables"`
			FolderAnalyzerAnalyzedFilesCount                  float64 `json:"FolderAnalyzer.analyzedFilesCount"`
			FolderAnalyzerScopeMilliseconds                   int     `json:"FolderAnalyzer.scopeMilliseconds"`
		} `json:"ScanDiagnostics"`
	} `json:"ScanMetadata"`
	AnalyzedFiles []struct {
		RelativePath string `json:"RelativePath"`
		Size         int    `json:"Size"`
		Fingerprints []struct {
			Type  string `json:"Type"`
			Value string `json:"Value"`
		} `json:"Fingerprints"`
	} `json:"AnalyzedFiles"`
	DependencyResolutionResults []DependencyResolution `json:"DependencyResolutionResults"`
	ContainerResolutionResults  struct {
		ImagePaths []interface{} `json:"ImagePaths"`
		Layers     struct {
		} `json:"Layers"`
	} `json:"ContainerResolutionResults"`
}

type DependencyResolution struct {
	Dependencies             []Dependency `json:"Dependencies"`
	PackageManagerFile       string       `json:"PackageManagerFile"`
	ResolvingModuleType      string       `json:"ResolvingModuleType"`
	DependencyResolverStatus string       `json:"DependencyResolverStatus"`
	Message                  string       `json:"Message"`
}

type Dependency struct {
	Children            []ID   `json:"Children"`
	ID                  ID     `json:"Id"`
	IsDirect            bool   `json:"IsDirect"`
	IsDevelopment       bool   `json:"IsDevelopment"`
	IsPluginDependency  bool   `json:"IsPluginDependency"`
	IsTestDependency    bool   `json:"IsTestDependency"`
	PotentialPrivate    bool   `json:"PotentialPrivate"`
	ResolvingModuleType string `json:"ResolvingModuleType"`
	AdditionalData      struct {
		ArtifactID string `json:"ArtifactId"`
		GroupID    string `json:"GroupId"`
	} `json:"AdditionalData"`
	TargetFrameworks []interface{} `json:"TargetFrameworks"`
}

type ID struct {
	NodeID  string `json:"NodeId"`
	Name    string `json:"Name"`
	Version string `json:"Version"`
}
