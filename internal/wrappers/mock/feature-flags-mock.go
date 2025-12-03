package mock

import (
	"fmt"

	"github.com/checkmarx/ast-cli/internal/wrappers"
)

var Flags wrappers.FeatureFlagsResponseModel
var Flag wrappers.FeatureFlagResponseModel
var FFErr error

type FeatureFlagsMockWrapper struct {
}

func (f FeatureFlagsMockWrapper) GetAll() (*wrappers.FeatureFlagsResponseModel, error) {
	fmt.Println("Called GetAll in FeatureFlagsMockWrapper")
	if len(Flags) == 0 {
		return &wrappers.FeatureFlagsResponseModel{Flag}, nil
	}
	return &Flags, nil
}

func (f FeatureFlagsMockWrapper) GetSpecificFlag(specificFlag string) (*wrappers.FeatureFlagResponseModel, error) {
	fmt.Println("Called GetSpecificFlag in FeatureFlagsMockWrapper with flag:", specificFlag)
	if FFErr != nil {
		fmt.Println(FFErr)
		return nil, FFErr
	}

	// If Flags (plural) is set, search for the specific flag by name
	if len(Flags) > 0 {
		for _, flag := range Flags {
			if flag.Name == specificFlag {
				fmt.Println("Returning flag from Flags collection:", flag.Status, "for flag name:", flag.Name)
				return &wrappers.FeatureFlagResponseModel{Name: flag.Name, Status: flag.Status}, nil
			}
		}
		// If flag not found in collection, return default (false)
		fmt.Println("Flag not found in Flags collection, returning default (false) for flag:", specificFlag)
		return &wrappers.FeatureFlagResponseModel{Name: specificFlag, Status: false}, nil
	}

	// Otherwise, return the single Flag (backward compatibility)
	fmt.Println("Returning flag:", Flag.Status, "for flag name:", Flag.Name)
	return &Flag, nil
}
