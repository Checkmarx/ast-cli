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
	var found = false
	var outString []byte
	err = json.Unmarshal([]byte(r.FileContent), &decoded)
	if err != nil {
		return "", err
	}
	dependencies := decoded.(map[string]interface{})["dependencies"]
	for key, element := range dependencies.(map[string]interface{}) {
		if key == r.PackageIdentifier {
			logger.PrintIfVerbose("Found package " + key + " with version " + element.(string) + ", replacing it with " + r.PackageVersion + ".")
			dependencies.(map[string]interface{})[key] = r.PackageVersion
			found = true
		}
	}
	decoded.(map[string]interface{})["dependencies"] = dependencies
	outString, err = json.MarshalIndent(decoded, "", "		")
	if err != nil {
		return "", err
	}
	if !found {
		return "", errors.Errorf("Package " + r.PackageIdentifier + " not found")
	}
	return string(outString), nil
}
