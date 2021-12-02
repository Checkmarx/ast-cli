package wrappers

import (
	"bytes"
	"encoding/json"
	"fmt"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/spf13/viper"
	"net/http"
)

type ResultsPredicatesHTTPWrapper struct {
	path        string
	contentType string
}

func NewResultsPredicatesHTTPWrapper(path string) ResultsPredicatesWrapper {
	return &ResultsPredicatesHTTPWrapper{
		path:        path,
		contentType: "application/json",
	}
}

func (r ResultsPredicatesHTTPWrapper) PredicateSeverityAndStateForSAST(predicate *Predicate) error {

	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	b := [...]Predicate{*predicate}
	jsonBytes, err := json.Marshal(b)
	if err != nil {
		return err
	}
	fmt.Println(r.path)
	fmt.Println(string(jsonBytes))

	//strTest := "[{\"similarityId\":\"1471101985\",\"projectId\":\"cc13fcbe-6684-4d66-ae9c-00a14fb8c063\",\"state\":\"to_verify\",\"comment\":\"\",\"severity\":\"medium\",\"scannerType\":\"sast\"}]"
	//c := []byte(strTest)


	resp, err2 := SendHTTPRequest(http.MethodPost, r.path, bytes.NewBuffer(jsonBytes), true, clientTimeout)
	fmt.Println("Test response")
	fmt.Println(resp)
	fmt.Println(err2)
	if err2 != nil {
		return err2
	}
	return nil
}






func (r ResultsPredicatesHTTPWrapper) PredicateSeverityAndStateForKICS(predicate *Predicate) error {
	panic("implement me")
}



