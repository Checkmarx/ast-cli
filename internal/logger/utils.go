package logger

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"strings"
	"unicode/utf8"

	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/spf13/viper"
)

var sanitizeFlags = []string{
	params.AstAPIKey, params.AccessKeyIDConfigKey, params.AccessKeySecretConfigKey,
	params.AstToken, params.SshValue,
	params.SCMTokenFlag,
}

func PrintIfVerbose(msg string) {
	if viper.GetBool(params.DebugFlag) {
		if utf8.Valid([]byte(msg)) {
			log.Print(sanitizeLogs(msg))
		} else {
			log.Print("Request contains binary data and cannot be printed!")
		}
	}
}

func PrintRequest(r *http.Request) {
	PrintIfVerbose("Sending API request to:")
	requestDump, err := httputil.DumpRequest(r, true)
	if err != nil {
		fmt.Println(err)
	}
	PrintIfVerbose(string(requestDump))
}

func PrintResponse(r *http.Response) {
	PrintIfVerbose("Receiving API response:")
	requestDump, err := httputil.DumpResponse(r, true)
	if err != nil {
		fmt.Println(err)
	}
	PrintIfVerbose(string(requestDump))
}

func sanitizeLogs(log string) string {
	if strings.Contains(log, "access_token") && strings.Contains(log, "token_type") {
		return ""
	}

	for _, flag := range sanitizeFlags {
		value := viper.GetString(flag)
		if len(value) > 0 {
			log = strings.ReplaceAll(log, value, "***")
		}
	}
	return log
}
