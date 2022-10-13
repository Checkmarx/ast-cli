package utils

import (
	"log"
	"testing"
)

func TestCleanURL_CleansCorrectly(t *testing.T) {
	uri := "https://codebashing.checkmarx.com/courses/java/////lessons/sql_injection/////"
	wantErr := false
	want := "https://codebashing.checkmarx.com/courses/java/lessons/sql_injection"
	got, err := CleanURL(uri)
	log.Println("error:", err)
	if (err != nil) != wantErr {
		t.Errorf("CleanURL() error = %v, wantErr %v", err, wantErr)
		return
	}
	if got != want {
		t.Errorf("CleanURL() got = %v, want %v", got, want)
	}
	log.Println("GOT:", got)
}

func TestCleanURL_invalid_URL_escape_error(t *testing.T) {
	uri := "#)@($_(*#_(*@$_))%(_#@_+#@$)$_$#@_@_##}^^^}!)(()!#@(`SPPSCOK^Ç^Ç`P$_$"
	wantErr := true
	want := ""
	got, err := CleanURL(uri)
	log.Println("error:", err)
	if (err != nil) != wantErr {
		t.Errorf("CleanURL() error = %v, wantErr %v", err, wantErr)
		return
	}
	if got != want {
		t.Errorf("CleanURL() got = %v, want %v", got, want)
	}
	log.Println("GOT:", got)
}
func TestCleanURL_cleans_correctly2(t *testing.T) {
	uri := "http://localhost:42/////test//test"
	wantErr := false
	want := "http://localhost:42/test/test"
	got, err := CleanURL(uri)
	log.Println("error:", err)
	if (err != nil) != wantErr {
		t.Errorf("CleanURL() error = %v, wantErr %v", err, wantErr)
		return
	}
	if got != want {
		t.Errorf("CleanURL() got = %v, want %v", got, want)
	}
	log.Println("GOT:", got)
}
