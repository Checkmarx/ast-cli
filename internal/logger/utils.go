package logger

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"strings"
	"unicode/utf8"

	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/spf13/viper"
)

const ContentLengthLimit = 1000000 // 1mb in bytes

var sanitizeFlags = []string{
	params.AstAPIKey, params.AccessKeyIDConfigKey, params.AccessKeySecretConfigKey,
	params.UsernameFlag, params.PasswordFlag,
	params.AstToken, params.SSHValue,
	params.SCMTokenFlag, params.ProxyKey,
	params.UploadURLEnv,
	params.SCSRepoTokenFlag,
	params.SCSRepoURLFlag,
}

func Print(msg string) {
	if utf8.Valid([]byte(msg)) {
		log.Print(sanitizeLogs(msg))
	} else {
		log.Print("Request contains binary data and cannot be printed!")
	}
}

func Printf(msg string, args ...interface{}) {
	Print(fmt.Sprintf(msg, args...))
}

func PrintIfVerbose(msg string) {
	if viper.GetBool(params.DebugFlag) || viper.GetString(params.LogFileFlag) != "" || viper.GetString(params.LogFileConsoleFlag) != "" {
		Print(msg)
	}
}

func PrintfIfVerbose(msg string, args ...interface{}) {
	PrintIfVerbose(fmt.Sprintf(msg, args...))
}

func PrintRequest(r *http.Request) {
	PrintIfVerbose("Sending API request to:")
	requestDump, err := httputil.DumpRequest(r, r.ContentLength < ContentLengthLimit)
	if err != nil {
		fmt.Println(err)
		return
	}
	PrintIfVerbose(string(requestDump))
}

func PrintResponse(r *http.Response, body bool) {
	PrintIfVerbose("Receiving API response:")
	requestDump, err := httputil.DumpResponse(r, body)
	if err != nil {
		fmt.Println(err)
		return
	}
	PrintIfVerbose(string(requestDump))
}

func sanitizeLogs(msg string) string {
	for _, flag := range sanitizeFlags {
		value := viper.GetString(flag)
		if len(value) > 0 {
			msg = strings.ReplaceAll(msg, value, "***")
		}
	}
	return msg
}

// SetOutput sets the output destination for the logger.
func SetOutput(w io.Writer) {
	log.SetOutput(w)
}
