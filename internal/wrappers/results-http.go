package wrappers

import (
	"encoding/json"
	"net/http"

	resultsHelpers "github.com/checkmarxDev/sast-results/pkg/web/helpers"

	"github.com/pkg/errors"
)

const (
	failedToParseGetResults = "Failed to parse list results"
)

type ResultsHTTPWrapper struct {
	path      string
	sastPath  string
	kicsPath  string
	scansPath string
}

func NewHTTPResultsWrapper(path, sastPath, kicsPath, scansPath string) ResultsWrapper {
	return &ResultsHTTPWrapper{
		path:      path,
		sastPath:  sastPath,
		kicsPath:  kicsPath,
		scansPath: scansPath,
	}
}

func (r *ResultsHTTPWrapper) GetScaAPIPath() string {
	return r.scansPath
}

func (r *ResultsHTTPWrapper) GetAllResultsByScanID(params map[string]string) (*ScanResultsCollection, *resultsHelpers.WebError, error) {
	params["limit"] = "10000"
	resp, err := SendHTTPRequestWithQueryParams(http.MethodGet, r.path, params, nil, DefaultTimeoutSeconds)
	if err != nil {
		return nil, nil, err
	}
	// TODO: REMOVE mocked decoder
	decoder := json.NewDecoder(resp.Body)
	//decoder := json.NewDecoder(strings.NewReader(mockResults))
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError:
		errorModel := resultsHelpers.WebError{}
		err = decoder.Decode(&errorModel)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseGetResults)
		}
		return nil, &errorModel, nil
	case http.StatusOK:
		model := ScanResultsCollection{}
		err = decoder.Decode(&model)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseGetResults)
		}
		return &model, nil, nil
	default:
		return nil, nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}

// Mock results
var mockResults = `
{
  "TotalCount": 7,
  "results": [        
    {
      "id": "12345",
      "similarityId": "-868420736",
      "vulnerabilityDetails": {
        "cweId": 602,
        "owasp2017": "A1"
      },
      "severity": "LOW",
      "firstScanId": "fc6a6e5e-3dab-4b3f-af2b-6dcf446626ef",
      "firstFoundAt": "2021-03-25T19:09:06Z",
      "foundAt": "2021-03-25T20:07:30Z",
      "status": "RECURRENT",
      "state": "NOT_EXPLOITABLE",
      "type": "sast",
      "data": {
        "queryId": 10526212270892872000,
        "queryName": "Client Side Only Validation",
        "group": "VbNet_Low_Visibility",
        "pathSystemId": "CF0SQeGPoCwKDphvpEFO5OUHZME=",
        "resultHash": "CF0SQeGPoCwKDphvpEFO5OUHZME=",
        "languageName": "VbNet",
        "nodes": [
          {
            "column": 15,
            "fileName": "ast_details_view.ts",
            "fullName": "src/ast_details_view.ts",
            "length": 14,
            "line": 1,
            "methodLine": 1,
            "name": "bookdetailpage",
            "domType": "ClassDecl"
          },
          {
            "column": 15,
            "fileName": "ast_details_view.ts",
            "fullName": "src/ast_details_view.ts",
            "length": 14,
            "line": 22,
            "methodLine": 1,
            "name": "bookdetailpage",
            "domType": "ClassDecl"            
          }
        ]
      },
      "comments": "This is long standing SASt error?"
    },
    {
      "id": "12345",
      "similarityId": "-868420736",
      "vulnerabilityDetails": {
        "cweId": 602,
        "owasp2017": "A1"
      },
      "severity": "LOW",
      "firstScanId": "fc6a6e5e-3dab-4b3f-af2b-6dcf446626ef",
      "firstFoundAt": "2021-03-25T19:09:06Z",
      "foundAt": "2021-03-25T20:07:30Z",
      "status": "NEW",
      "state": "NOT_EXPLOITABLE",
      "type": "sast",
      "data": {
        "queryId": 10526212270892872000,
        "queryName": "Jeff Major Issue",
        "group": "VbNet_Low_Visibility",
        "pathSystemId": "CF0SQeGPoCwKDphvpEFO5OUHZME=",
        "resultHash": "CF0SQeGPoCwKDphvpEFO5OUHZME=",
        "languageName": "Java",
        "nodes": [
          {
            "column": 15,
            "fileName": "ast_results_provider.ts",
            "fullName": "src/OtherItem/ast_results_provider.ts",
            "length": 25,
            "line": 68,
            "methodLine": 1,
            "name": "bookdetailpage",
            "domType": "ClassDecl",
            "nodeSystemId": "fTPHOKt18pwXgBGUaMx8XV7rL5s=",
            "nodeHash": "fTPHOKt18pwXgBGUaMx8XV7rL5s="
          }
        ]
      },
      "comments": "This is long standing SASt error?"
    },
    {
      "id": "12345",
      "similarityId": "-868420736",
      "vulnerabilityDetails": {
        "cweId": 602,
        "owasp2017": "A1"
      },
      "severity": "HIGH",
      "firstScanId": "fc6a6e5e-3dab-4b3f-af2b-6dcf446626ef",
      "firstFoundAt": "2021-03-25T19:09:06Z",
      "foundAt": "2021-03-25T20:07:30Z",
      "status": "NEW",
      "state": "NOT_EXPLOITABLE",
      "type": "sast",
      "data": {
        "queryId": 10526212270892872000,
        "queryName": "SQL Injection",
        "group": "VbNet_Low_Visibility",
        "pathSystemId": "CF0SQeGPoCwKDphvpEFO5OUHZME=",
        "resultHash": "CF0SQeGPoCwKDphvpEFO5OUHZME=",
        "languageName": "VbNet",
        "nodes": [
          {
            "column": 15,
            "fileName": "test.php",
            "fullName": "src/test.php",
            "length": 6,
            "line": 90,
            "methodLine": 1,
            "name": "bookdetailpage",
            "domType": "ClassDecl",
            "nodeSystemId": "fTPHOKt18pwXgBGUaMx8XV7rL5s=",
            "nodeHash": "fTPHOKt18pwXgBGUaMx8XV7rL5s="
          }
        ]
      },
      "comments": "This another error we created for testing."
    },
    {
      "id": "12345",
      "similarityId": "-868420736",
      "vulnerabilityDetails": {
        "cweId": 602,
        "owasp2017": "A1"
      },
      "severity": "MEDIUM",
      "firstScanId": "fc6a6e5e-3dab-4b3f-af2b-6dcf446626ef",
      "firstFoundAt": "2021-03-25T19:09:06Z",
      "foundAt": "2021-03-25T20:07:30Z",
      "status": "RECURRENT",
      "state": "NOT_EXPLOITABLE",
      "type": "sast",
      "data": {
        "queryId": 10526212270892872000,
        "queryName": "XSS",
        "group": "VbNet_Low_Visibility",
        "pathSystemId": "CF0SQeGPoCwKDphvpEFO5OUHZME=",
        "resultHash": "CF0SQeGPoCwKDphvpEFO5OUHZME=",
        "languageName": "VbNet",
        "nodes": [
          {
            "column": 15,
            "fileName": "ast_details_view.ts",
            "fullName": "src/ast_details_view.ts",
            "length": 32,
            "line": 44,
            "methodLine": 1,
            "name": "bookdetailpage",
            "domType": "ClassDecl",
            "nodeSystemId": "fTPHOKt18pwXgBGUaMx8XV7rL5s=",
            "nodeHash": "fTPHOKt18pwXgBGUaMx8XV7rL5s="
          },
          {
            "column": 15,
            "fileName": "ast_results_provider.ts",
            "fullName": "src/OtherItem/ast_results_provider.ts",
            "length": 19,
            "line": 44,
            "methodLine": 1,
            "name": "bookdetailpage",
            "domType": "ClassDecl",
            "nodeSystemId": "fTPHOKt18pwXgBGUaMx8XV7rL5s=",
            "nodeHash": "fTPHOKt18pwXgBGUaMx8XV7rL5s="
          },
          {
            "column": 15,
            "fileName": "test.php",
            "fullName": "src/test.php",
            "length": 11,
            "line": 44,
            "methodLine": 1,
            "name": "bookdetailpage",
            "domType": "ClassDecl",
            "nodeSystemId": "fTPHOKt18pwXgBGUaMx8XV7rL5s=",
            "nodeHash": "fTPHOKt18pwXgBGUaMx8XV7rL5s="
          }
        ]
      },
      "comments": "The alternative test page."
    },

        {
          "id": "12346",
          "type": "dependency",
          "similarityId": "42",
          "vulnerabilityMetadata": {
            "cvssScore": 7.5,
            "cveName": "CVE-2014-0114",
            "cweId": 20,
            "cvss*": "any cvss calc values"
          },
          "severity": "INFO",
          "firstScanId": "fc6a6e5e-3dab-4b3f-af2b-6dcf446626ef",
          "firstFoundAt": "2021-03-25T19:09:06Z",
          "foundAt": "2021-03-25T20:07:30Z",
          "status": "RECURRENT",
          "state": "CONFIRMED",          
          "data": {
            "description": "Apache Commons BeanUtils, as distributed in lib/commons-beanutils-1.8.0.jar...",
            "recommendations": "",
            "packageId": "Maven-commons-beanutils:commons-beanutils-1.8.3",
            "recommendedVersion": "1.9.4",
            "exploitableMethods": [
              ""
            ],
            "packagePublishDate": "2014-04-30T10:49:00Z",
            "packageData": [
              {
                "url": "https://issues.apache.org/jira/browse/BEANUTILS-520",
                "type": "Issue",
                "comment": "Apache Commons BeanUtils"
              },
              {
                "url": "https://github.com/apache/commons-beanutils/pull/7",
                "type": "Pull request",
                "comment": ""
              }
            ]
          },
          "comments": "href to comments?"
        },    
        {
          "id": "12347",
          "similarityId": "-1",
          "vulnerabilityDetails": {
            "royaltyFree": "Free",
            "copyrightRiskScore": "3",
            "linking": "NonViral",
            "copyLeft": "NoCopyleft",
            "patentRiskScore": "3"
          },
          "severity": "LOW",
          "firstScanId": "fc6a6e5e-3dab-4b3f-af2b-6dcf446626ef",
          "firstFoundAt": "2021-03-25T19:09:06Z",
          "foundAt": "2021-03-25T20:07:30Z",
          "status": "RECURRENT",
          "state": "CONFIRMED",
          "type": "license",
          "data": {
            "queryId": "Unknown-abbrev-1.0.9-ISC",
            "queryName": "ISC",
            "queryUrl": "https://opensource.org/licenses/ISC",
            "packageType": "Npm",
            "packageUrl": "https://www.npmjs.com/package/abbrev/v/1.0.9"
          },
          "comments": "href to comments?"
        },
        {
          "id": "12348",
          "type": "infrastructure",
          "similarityId": "80c80ca05c3cd6fdddc808e042d3a404aee120a7419d89649c909409d6235614",
          "vulnerabilityDetails": {
            "tbd": "tbd"
          },
          "severity": "MEDIUM",
          "firstScanId": "fc6a6e5e-3dab-4b3f-af2b-6dcf446626ef",
          "firstFoundAt": "2021-03-25T19:09:06Z",
          "foundAt": "2021-03-25T20:07:30Z",
          "status": "RECURRENT",
          "state": "NOT_EXPLOITABLE",
          "data": {
            "queryId": "a3a055d2-9a2e-4cc9-b9fb-12850a1a3a4b",
            "queryName": "AD Admin Not Configured For SQL Server",
            "group": "Build Process",
            "queryUrl": "https://docs.docker.com/engine/reference/builder/#entrypoint",
            "fileName": "/terraform/azure/sql.tf",
            "line": 9,
            "platform": "Terraform",
            "issueType": "IncorrectValue",
            "searchKey": "FROM={{alpine:3.12.0}}.{{CMD /entrypoint.sh && crond -l 2 -f}}",
            "searchValue": "",
            "expectedValue": "FROM={{alpine:3.12.0}}.{{CMD /entrypoint.sh && crond -l 2 -f}} is in the JSON Notation",
            "actualValue": "FROM={{alpine:3.12.0}}.{{CMD /entrypoint.sh && crond -l 2 -f}} isn't in the JSON Notation",
            "value": null,
            "description": "Ensure that we are using JSON in the CMD and ENTRYPOINT Arguments"
          },
          "comments": "href to comments?"
        }
  ]
}

`
