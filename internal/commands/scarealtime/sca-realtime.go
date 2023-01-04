package scarealtime

import (
	"encoding/json"
	"io/ioutil"
	"strconv"
	"time"

	"fmt"
	"os/exec"

	commonParams "github.com/checkmarx/ast-cli/internal/params"

	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type ScaResultsFile struct {
	ScanMetadata struct {
		StartTime     time.Time `json:"StartTime"`
		ScanPath      string    `json:"ScanPath"`
		ScanArguments struct {
			ProjectDownloadURL interface{} `json:"ProjectDownloadUrl"`
			ScanID             string      `json:"ScanId"`
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

// RunScaRealtime Main method responsible to run sca realtime feature
func RunScaRealtime(scaRealTimeWrapper wrappers.ScaRealTimeWrapper) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		projectDirPath, _ := cmd.Flags().GetString(commonParams.ScaRealtimeProjectDir)
		if projectDirPath == "" {
			return errors.New("missing project path")
		}

		fmt.Println("Handling SCA Resolver...")
		scaResolverExecutableFile, err := getScaResolver()
		if err != nil {
			return err
		}

		err = executeSCAResolver(scaResolverExecutableFile, projectDirPath)
		if err != nil {
			return err
		}

		err = buildSCAResults(scaRealTimeWrapper)
		if err != nil {
			return err
		}

		return nil
	}
}

// executeSCAResolver Executes sca resolver for a specific path
func executeSCAResolver(executable, projectPath string) error {
	args := []string{
		"offline",
		"-s",
		projectPath,
		"-n",
		"dev_sca_realtime_project",
		"-r",
		scaResolverWorkingDir + "/cx-sca-realtime-results.json",
	}
	fmt.Println(fmt.Printf("Running SCA resolver with args: %v", args))

	_, err := exec.Command(executable, args...).Output()
	if err != nil {
		return err
	}
	fmt.Println("SCA Resolver finished successfully!")

	return nil
}

func buildSCAResults(scaRealTimeWrapper wrappers.ScaRealTimeWrapper) error {
	file, err := ioutil.ReadFile(scaResolverWorkingDir + "/cx-sca-realtime-results.json")
	if err != nil {
		return err
	}

	data := ScaResultsFile{}
	_ = json.Unmarshal(file, &data)

	var modelResults []wrappers.ScaVulnerabilitiesResponseModel

	for _, f := range data.DependencyResolutionResults {
		dependencyMap := make(map[string]wrappers.ScaDependencyBodyRequest)

		for _, dependencyResolution := range f.Dependencies {
			dependencyMap[dependencyResolution.ID.NodeID] = wrappers.ScaDependencyBodyRequest{
				PackageName:    dependencyResolution.ID.Name,
				Version:        dependencyResolution.ID.Version,
				PackageManager: dependencyResolution.ResolvingModuleType,
			}
			if len(dependencyResolution.Children) > 0 {
				for _, dependencyChildren := range dependencyResolution.Children {
					dependencyMap[dependencyResolution.ID.NodeID] = wrappers.ScaDependencyBodyRequest{
						PackageName:    dependencyChildren.Name,
						Version:        dependencyChildren.Version,
						PackageManager: dependencyResolution.ResolvingModuleType,
					}
				}
			}
		}

		var body []wrappers.ScaDependencyBodyRequest
		for _, value := range dependencyMap {
			body = append(body, value)
		}

		model, _, _ := scaRealTimeWrapper.GetScaVulnerabilitiesPackages(body)
		for _, value := range model {
			value.FileName = f.PackageManagerFile
			modelResults = append(modelResults, value)
		}
	}

	err = convertToScanResults(modelResults)
	if err != nil {
		return err
	}

	return nil
}

func convertToScanResults(data []wrappers.ScaVulnerabilitiesResponseModel) error {
	var results []*wrappers.ScanResult

	for _, packageData := range data {
		for _, vulnerability := range packageData.Vulnerabilities {
			score, _ := strconv.ParseFloat(vulnerability.Cvss3.BaseScore, 8)

			results = append(results, &wrappers.ScanResult{
				Type:        vulnerability.Type,
				ScaType:     "vulnerability",
				Description: vulnerability.Description,
				Severity:    vulnerability.Severity,
				VulnerabilityDetails: wrappers.VulnerabilityDetails{
					CweID:     vulnerability.Cve,
					CvssScore: score,
					CveName:   vulnerability.Cve,
					CVSS: wrappers.VulnerabilityCVSS{
						Version:            vulnerability.VulnerabilityVersion,
						AttackVector:       vulnerability.Cvss3.AttackVector,
						Availability:       vulnerability.Cvss3.Availability,
						Confidentiality:    vulnerability.Cvss3.Confidentiality,
						AttackComplexity:   vulnerability.Cvss3.AttackComplexity,
						IntegrityImpact:    vulnerability.Cvss3.Integrity,
						Scope:              vulnerability.Cvss3.Scope,
						PrivilegesRequired: vulnerability.Cvss3.PrivilegesRequired,
						UserInteraction:    vulnerability.Cvss3.UserInteraction,
					},
				},
				ScanResultData: wrappers.ScanResultData{
					PackageData: vulnerability.References,
					ScaPackageCollection: &wrappers.ScaPackageCollection{
						FixLink: "https://devhub.checkmarx.com/cve-details/" + vulnerability.Cve,
					},
					Nodes: []*wrappers.ScanResultNode{{
						FileName: packageData.FileName,
					}},
				},
			})
		}
	}

	resultsCollection := wrappers.ScanResultsCollection{
		Results:    results,
		TotalCount: uint(len(results)),
	}

	resultsJSON, errs := json.Marshal(resultsCollection)
	if errs != nil {
		return errors.Errorf("%s", errs)
	}
	fmt.Println(string(resultsJSON))

	return nil
}
