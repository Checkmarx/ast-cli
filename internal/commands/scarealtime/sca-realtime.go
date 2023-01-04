package scarealtime

import (
	"encoding/json"
	"io/ioutil"
	"strconv"
	"strings"

	"fmt"
	"os/exec"

	commonParams "github.com/checkmarx/ast-cli/internal/params"

	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// RunScaRealtime Main method responsible to run sca realtime feature
func RunScaRealtime(scaRealTimeWrapper wrappers.ScaRealTimeWrapper) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		fmt.Println("Handling SCA Resolver...")

		err := downloadSCAResolverAndHashFileIfNeeded(&Params)
		if err != nil {
			return err
		}

		projectDirPath, _ := cmd.Flags().GetString(commonParams.ScaRealtimeProjectDir)
		err = executeSCAResolver(projectDirPath)
		if err != nil {
			return err
		}

		err = getSCAResults(scaRealTimeWrapper)
		if err != nil {
			return err
		}

		return nil
	}
}

// executeSCAResolver Executes sca resolver for a specific path
func executeSCAResolver(projectPath string) error {
	args := []string{
		"offline",
		"-s",
		projectPath,
		"-n",
		"dev_sca_realtime_project",
		"-r",
		ScaResolverWorkingDir + "/cx-sca-realtime-results.json",
	}
	fmt.Println(fmt.Printf("Running SCA resolver with args: %v", args))

	_, err := exec.Command(Params.ExecutableFilePath, args...).Output()
	if err != nil {
		return err
	}
	fmt.Println("SCA Resolver finished successfully!")

	return nil
}

func getSCAResults(scaRealTimeWrapper wrappers.ScaRealTimeWrapper) error {
	file, err := ioutil.ReadFile(ScaResolverWorkingDir + "/cx-sca-realtime-results.json")
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
				Label:       commonParams.ScaType,
				Description: vulnerability.Description,
				Severity:    strings.ToUpper(vulnerability.Severity),
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
					PackageIdentifier: packageData.PackageName,
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
