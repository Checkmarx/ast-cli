package mock

import (
	"fmt"

	"github.com/checkmarx/ast-cli/internal/wrappers/grpcs"
)

var (
	specialErrorPortNumber = 1
)

type VorpalMockWrapper struct {
	Port int
}

func NewVorpalMockWrapper(port int) *VorpalMockWrapper {
	return &VorpalMockWrapper{Port: port}
}

func (v *VorpalMockWrapper) Scan(fileName, sourceCode string) (*grpcs.ScanResult, error) {
	if fileName == "csharp-no-vul.cs" {
		return ReturnFailureResponseMock(), nil
	}
	return ReturnSuccessfulResponseMock(), nil
}

func (v *VorpalMockWrapper) HealthCheck() error {
	if v.Port == specialErrorPortNumber {
		return fmt.Errorf("error %d", InternalError)
	}
	return nil
}

func (v *VorpalMockWrapper) ShutDown() error {
	return nil
}

func (v *VorpalMockWrapper) GetPort() int {
	return v.Port
}

func ReturnSuccessfulResponseMock() *grpcs.ScanResult {
	return &grpcs.ScanResult{
		RequestID: "1234567890",
		Status:    true,
		Message:   "Scan completed successfully.",
		ScanDetails: []grpcs.ScanDetail{
			{
				Language:    "Python",
				RuleName:    "Stored XSS",
				Severity:    "High",
				FileName:    "python-vul-file.py",
				Line:        37,
				Remediation: "Fully encode all dynamic data, regardless of source, before embedding it in output.",
				Description: "The method undefined embeds untrusted data in generated output with write, at line 80 of /python-vul-file.py." +
					"This untrusted data is embedded into the output without proper sanitization or encoding, enabling an attacker to inject malicious code into the generated web-page." +
					"The attacker would be able to alter the returned web page by saving malicious data in a data-store ahead of time." +
					"The attacker's modified data is then read from the database by the undefined method with read, at line 37 of /python-vul-file.py." +
					"This untrusted data then flows through the code straight to the output web page, without sanitization.  This can enable a Stored Cross-Site Scripting (XSS) attack.",
			},
			{
				Language:    "Python",
				RuleName:    "Missing HSTS Header",
				Severity:    "Medium",
				FileName:    "python-vul-file.py",
				Line:        76,
				Remediation: "Before setting the HSTS header - consider the implications it may have: Forcing HTTPS will prevent any future use of HTTP",
				Description: "The web-application does not define an HSTS header, leaving it vulnerable to attack.",
			},
		},
	}
}

func ReturnFailureResponseMock() *grpcs.ScanResult {
	return &grpcs.ScanResult{
		RequestID: "some-request-id",
		Status:    false,
		Message:   "Scan failed.",
		Error:     &grpcs.Error{Code: InternalError, Description: "An internal error occurred."},
	}
}

func (v *VorpalMockWrapper) ConfigurePort(port int) {

}

const (
	UnknownError   = 0
	InvalidRequest = 1
	InternalError  = 2
)
