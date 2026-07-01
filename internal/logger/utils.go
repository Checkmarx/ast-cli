package logger

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/spf13/viper"
)

const ContentLengthLimit = 1000000 // 1mb in bytes

// logTimestampLayout is the date/time layout for log records. Go's stdlib log
// flags can only emit slash-separated, fixed-format timestamps, so we disable
// them (log.SetFlags(0)) and prepend our own UTC, ISO-8601-style stamp via
// timestampedWriter — producing e.g. "2026-06-30 14:23:01 UTC".
const logTimestampLayout = "2006-01-02 15:04:05"

// timestampedWriter prepends a UTC timestamp to each log record. The stdlib log
// package issues exactly one Write per log call, so each line gets one stamp.
type timestampedWriter struct {
	w io.Writer
}

func (tw *timestampedWriter) Write(p []byte) (int, error) {
	prefix := time.Now().UTC().Format(logTimestampLayout) + " UTC "
	if _, err := io.WriteString(tw.w, prefix); err != nil {
		return 0, err
	}
	return tw.w.Write(p)
}

// init disables the stdlib log timestamp (so it isn't duplicated) and routes the
// default logger through timestampedWriter on stderr. This covers --debug console
// output, where SetOutput is never called.
func init() {
	log.SetFlags(0)
	log.SetOutput(&timestampedWriter{w: os.Stderr})
}

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

// SetOutput sets the output destination for the logger, wrapping it so every
// record is prefixed with a UTC timestamp (see timestampedWriter).
func SetOutput(w io.Writer) {
	log.SetOutput(&timestampedWriter{w: w})
}
