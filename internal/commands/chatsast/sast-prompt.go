package chatsast

import (
	"fmt"
	"os"
	"strings"
)

const promptTemplate = `Checkmarx Static Application Security Testing (SAST) detected the %s vulnerability within the provided %s code snippet. 
The attack vector is presented by code snippets annotated by comments in the form ` + "`//SAST Node #X: element (element-type)`" + ` where X is 
the node index in the result, ` + "`element`" + ` is the name of the element through which the data flows, and the ` + "`element-type`" + ` is it's type. 
The first and last nodes are indicated by ` + "`(input ...)` and `(output ...)`" + ` respectively:
` + "```" + `
%s
` + "```" + `
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

Next, please provide code that fixes the vulnerability so that a developer can copy paste instead of the snippet above.
 
Your analysis should be presented in the following format:
    CONFIDENCE: num
    EXPLANATION: short_text
    FIX: fixed_snippet`

func CreatePrompt(result Result, sources map[string][]string) (string, error) {
	promptSource, err := createSourceForPrompt(result, sources)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(promptTemplate, result.Data.QueryName, result.Data.LanguageName, promptSource), nil
}

func createSourceForPrompt(result Result, sources map[string][]string) (string, error) {
	var sourcePrompt []string
	methodsInPrompt := make(map[string][]string)
	for j, node := range result.Data.Nodes {
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
		if j == 0 {
			edge = " (input)"
		} else if j == len(result.Data.Nodes)-1 {
			edge = " (output)"
		} else {
			edge = ""
		}
		methodLines[lineInMethod] += fmt.Sprintf("//SAST Node #%d%s: %s (%s)", j, edge, node.Name, node.DomType)
		methodsInPrompt[sourceFilename+":"+node.Method] = methodLines
	}

	for _, methodLines := range methodsInPrompt {
		methodLines = append(methodLines, "// method continues ...")
		sourcePrompt = append(sourcePrompt, methodLines...)
	}

	return strings.Join(sourcePrompt, "\n"), nil
}

func createSourceForPromptWithFullMethod(result Result, sources map[string][]string) (string, error) {
	var sourcePrompt []string
	methodsInPrompt := make(map[string][]string)
	for j, node := range result.Data.Nodes {
		sourceFilename := strings.ReplaceAll(node.FileName, "\\", "/")
		methodLines, exists := methodsInPrompt[sourceFilename+":"+node.Method]
		if !exists {
			m, err := GetFullMethodByBracketBalancing(sourceFilename, sources[sourceFilename], node.MethodLine)
			methodLines = m
			if err != nil {
				return "", fmt.Errorf("error getting full method %s: %v", node.Method, err)
			}
		}
		lineInMethod := node.Line - node.MethodLine
		var edge string
		if j == 0 {
			edge = " (input)"
		} else if j == len(result.Data.Nodes)-1 {
			edge = " (output)"
		} else {
			edge = ""
		}
		methodLines[lineInMethod] += fmt.Sprintf("//SAST Node #%d%s: %s (%s)", j, edge, node.Name, node.DomType)
		methodsInPrompt[sourceFilename+":"+node.Method] = methodLines
	}

	for _, methodLines := range methodsInPrompt {
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

// The following function can be used to extract a method by balancing braces.
// TODO: fix bug where brackets in comments and string literals are counted
func GetFullMethodByBracketBalancing(filename string, lines []string, lineNumber int) ([]string, error) {
	if lineNumber < 1 || lineNumber > len(lines) {
		return nil, fmt.Errorf("line number %d is out of range", lineNumber)
	}

	// Adjust line number to 0-based index for slice access
	startIndex := lineNumber - 1

	// Find the opening brace of the method
	braceCount := 0
	firstBraceFound := false
	var methodLines []string
	for i := startIndex; i < len(lines); i++ {
		l := lines[i]
		s := ""
		if i == startIndex {
			s = fmt.Sprintf("// %s:%d", filename, lineNumber)
		}
		line := l + s
		methodLines = append(methodLines, line)

		j := strings.IndexAny(line, "{}")
		for j > -1 {
			if line[j] == '{' {
				firstBraceFound = true
				braceCount++
			} else if line[j] == '}' {
				braceCount--
			}
			line = line[j+1:]
			j = strings.IndexAny(line, "{}")
		}
		// If all braces are closed, the method ends
		if braceCount == 0 && firstBraceFound {
			return methodLines, nil
		}

		if braceCount < 0 {
			return nil, fmt.Errorf("too many closing braces found for method starting at line %d", lineNumber)
		}
	}
	return nil, fmt.Errorf("no matching closing brace found for method starting at line %d", lineNumber)
}

// read the prompt template text file from filename
func ReadPromptTemplate(filename string) (string, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
