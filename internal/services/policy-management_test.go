package services

import (
	"fmt"
	"testing"

	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestHandlePolicyEvaluation(t *testing.T) {
	type args struct {
		cmd           *cobra.Command
		policyWrapper wrappers.PolicyWrapper
		scan          *wrappers.ScanResponseModel
		agent         string
	}
	tests := []struct {
		name    string
		args    args
		want    *wrappers.PolicyResponseModel
		wantErr assert.ErrorAssertionFunc
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := HandlePolicyEvaluation(tt.args.cmd, tt.args.policyWrapper, tt.args.scan)
			if !tt.wantErr(t, err, fmt.Sprintf("HandlePolicyEvaluation(%v, %v, %v, %v)", tt.args.cmd, tt.args.policyWrapper, tt.args.scan, tt.args.agent)) {
				return
			}
			assert.Equalf(t, tt.want, got, "HandlePolicyEvaluation(%v, %v, %v, %v)", tt.args.cmd, tt.args.policyWrapper, tt.args.scan, tt.args.agent)
		})
	}
}
