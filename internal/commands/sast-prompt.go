package commands

import (
	"fmt"
	"regexp"
	"strings"
)

const systemPrompt = `You are the Checkmarx AI Guided Remediation bot who can answer technical questions related to the results of Checkmarx Static Application 
Security Testing (SAST). You should be able to analyze and understand both the technical aspects of the security results and the common queries users may have 
about the results. You should also be capable of delivering clear, concise, and informative answers to help take appropriate action based on the findings.
If a question irrelevant to the mentioned source code or SAST result is asked, answer 'I am the AI Guided Remediation assistant and can answer only on questions 
related to source code or SAST results or SAST Queries'.`

const (
	confidence  = "CONFIDENCE:"
	explanation = "EXPLANATION:"
	fix         = "FIX"
	code        = "```"
)

const userPromptTemplate = `Checkmarx Static Application Security Testing (SAST) detected the %s vulnerability within the provided %s code snippet. 
The attack vector is presented by code snippets annotated by comments in the form ` + "`//SAST Node #X: element (element-type)`" + ` where X is 
the node index in the result, ` + "`element`" + ` is the name of the element through which the data flows, and the ` + "`element-type`" + ` is it's type. 
The first and last nodes are indicated by ` + "`(input ...)` and `(output ...)`" + ` respectively:
` + code + `
%s
` + code + `
Please review the code above and provide a confidence score ranging from 0 to 100. 
A score of 0 means you believe the result is completely incorrect, unexploitable, and a false positive. 
A score of 100 means you believe the result is completely correct, exploitable, and a true positive.
 
Instructions for confidence score computation:
 
1. The confidence score of a vulnerability which can be done from the Internet is much higher than from the local console.
2. The confidence score of a vulnerability which can be done by anonymous user is much higher than of an authenticated user.
3. The confidence score of a vulnerability with a vector starting with a stored input (like from files/db etc) cannot be more than 50. 
This is also known as a second-order vulnerability
4. Pay your special attention to the first and last code snippet - whether a specific vulnerability found by Checkmarx SAST can start/occur here, 
or it's a false positive.
5. If you don't find enough evidence about a vulnerability, just lower the score.
6. If you are not sure, just lower the confidence - we don't want to have false positive results with a high confidence score.
 
Please provide a brief explanation for your confidence score, don't mention all the instruction above.

Next, please provide code that remediates the vulnerability so that a developer can copy paste instead of the snippet above.
 
Your analysis should be presented in the following format:
` + confidence + `number
` + explanation + `short_text
` + fix + `: fixed_snippet`

func GetSystemPrompt() string {
	return systemPrompt
}

func CreateUserPrompt(result *Result, sources map[string][]string) (string, error) {
	promptSource, err := createSourceForPrompt(result, sources)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(userPromptTemplate, result.Data.QueryName, result.Data.LanguageName, promptSource), nil
}

func createSourceForPrompt(result *Result, sources map[string][]string) (string, error) {
	var sourcePrompt []string
	methodsInPrompt := make(map[string][]string)
	for i := range result.Data.Nodes {
		node := result.Data.Nodes[i]
		sourceFilename := strings.ReplaceAll(node.FileName, "\\", "/")
		methodLines, exists := methodsInPrompt[sourceFilename+":"+node.Method]
		if !exists {
			m, err := GetMethodByMethodLine(sourceFilename, sources[sourceFilename], node.MethodLine, node.Line, false)
			methodLines = m
			if err != nil {
				return "", fmt.Errorf("error getting method %s: %v", node.Method, err)
			}
		} else if len(methodLines) < node.Line-node.MethodLine+1 {
			m, err := GetMethodByMethodLine(sourceFilename, sources[sourceFilename], node.MethodLine, node.Line, true)
			methodLines = m
			if err != nil {
				return "", fmt.Errorf("error getting method %s: %v", node.Method, err)
			}
		}
		lineInMethod := node.Line - node.MethodLine
		var edge string
		if i == 0 {
			edge = " (input)"
		} else if i == len(result.Data.Nodes)-1 {
			edge = " (output)"
		} else {
			edge = ""
		}

		// change UnknownReference to something more informational like VariableReference or TypeNameReference
		nodeType := node.DomType
		if node.DomType == "UnknownReference" {
			if node.TypeName == "" {
				nodeType = "VariableReference"
			} else {
				nodeType = node.TypeName + "Reference"
			}
		}
		methodLines[lineInMethod] += fmt.Sprintf("//SAST Node #%d%s: %s (%s)", i, edge, node.Name, nodeType)
		methodsInPrompt[sourceFilename+":"+node.Method] = methodLines
	}
	for _, methodLines := range methodsInPrompt {
		methodLines = append(methodLines, "// method continues ...")
		sourcePrompt = append(sourcePrompt, methodLines...)
	}

	return strings.Join(sourcePrompt, "\n"), nil
}

func GetMethodByMethodLine(filename string, lines []string, methodLineNumber, nodeLineNumber int, tagged bool) ([]string, error) {
	if methodLineNumber < 1 || methodLineNumber > len(lines) {
		return nil, fmt.Errorf("method line number %d is out of range", methodLineNumber)
	}

	if nodeLineNumber < 1 || nodeLineNumber > len(lines) {
		return nil, fmt.Errorf("node line number %d is out of range", nodeLineNumber)
	}

	if nodeLineNumber < methodLineNumber {
		return nil, fmt.Errorf("node line number %d is less than method line number %d", nodeLineNumber, methodLineNumber)
	}

	// Adjust line number to 0-based index for slice access
	startIndex := methodLineNumber - 1
	numberOfLines := nodeLineNumber - methodLineNumber + 1
	methodLines := lines[startIndex : startIndex+numberOfLines]
	if !tagged {
		methodLines[0] += fmt.Sprintf("// %s:%d", filename, methodLineNumber)
	}
	return methodLines, nil
}

func AddNewlinesIfNecessary(responseContent []string) []string {
	if len(responseContent) == 0 {
		return responseContent
	}
	stringToFix := responseContent[len(responseContent)-1]

	stringToFix = addNewlineIfNecessary(stringToFix, confidence, explanation)
	stringToFix = addNewlineIfNecessary(stringToFix, explanation, fix)
	return append(responseContent[:len(responseContent)-1], stringToFix)
}

func addNewlineIfNecessary(s, from, to string) string {
	startsAt := strings.Index(s, from) + len(from)
	upTo := strings.Index(s, to)
	if startsAt == -1 || upTo == -1 {
		return s
	}
	if !endsWithNewlineAndWhitespace(s[startsAt:upTo]) {
		return s[:upTo] + "\n" + s[upTo:]
	}
	return s
}

func endsWithNewlineAndWhitespace(s string) bool {
	// Compile the regular expression that matches a newline followed by
	// zero or more whitespace characters at the end of the string.
	re := regexp.MustCompile(`\n\s*$`)
	// Use the FindString method to find a match. If a match is found,
	// it means the string ends with a newline and possibly other whitespace characters.
	return re.FindString(s) != ""
}
