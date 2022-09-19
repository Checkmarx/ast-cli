package remediation

import (
	"encoding/json"

	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/pkg/errors"
)

func (r PackageContentJSON) Parser() (string, error) {
	// Needs to be a generic interface to read the entire file
	var decoded interface{}
	var err error
	var found bool
	var foundInDev bool
	var outString []byte
	err = json.Unmarshal([]byte(r.FileContent), &decoded)
	if err != nil {
		return "", err
	}
	found, decoded.(map[string]interface{})["dependencies"] = replace(
		decoded.(map[string]interface{})["dependencies"],
		r,
	)
	foundInDev, decoded.(map[string]interface{})["devDependencies"] = replace(
		decoded.(map[string]interface{})["devDependencies"],
		r,
	)
	outString, err = json.MarshalIndent(decoded, "", "  ")
	if err != nil {
		return "", err
	}
	if !(found || foundInDev) {
		return "", errors.Errorf("Package " + r.PackageIdentifier + " not found")
	}
	return string(outString), nil
}

func replace(dependencies interface{}, r PackageContentJSON) (
	dependencyFound bool,
	dependencyMap map[string]interface{},
) {
	var found = false
	if dependencies != nil {
		dependencyMap = dependencies.(map[string]interface{})
		for key, element := range dependencyMap {
			if key == r.PackageIdentifier {
				logger.PrintIfVerbose("Found package " + key + " with version " + element.(string) + ", replacing it with " + r.PackageVersion + ".")
				dependencyMap[key] = r.PackageVersion
				found = true
			}
		}
	}
	return found, dependencyMap
}
