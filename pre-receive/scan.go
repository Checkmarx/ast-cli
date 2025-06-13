package pre_receive

import (
	"bufio"
	"fmt"
	"github.com/Checkmarx/secret-detection/pkg/report"
	"github.com/checkmarx/2ms/lib/reporting"
	"github.com/checkmarx/2ms/lib/secrets"
	twoms "github.com/checkmarx/2ms/pkg"
	"github.com/checkmarx/2ms/plugins"
	"github.com/gitleaks/go-gitdiff/gitdiff"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type ScanConfig struct {
	ignoreResultIds []string
	ignoreRules     []string
	pathExclusion   []string
}

const (
	zeroRev             = "0000000000000000000000000000000000000000"
	unknownCommit       = "unknown"
	gitPushOptionPrefix = "GIT_PUSH_OPTION_"
	gitPushOptionCount  = "GIT_PUSH_OPTION_COUNT"
	skipScanKeyword     = "skip-secret-scanner"
)

func Scan(configPath string, logsPath string) error {
	refs, err := readRefsFromStdin()
	if err != nil {
		return fmt.Errorf("reading pushed refs: %w", err)
	}

	if skipScan() {
		fmt.Print("Cx Secret Scanner bypassed")
		err = logSkip(logsPath, refs)
		if err != nil {
			return err
		}
		return nil
	}

	scanConfig, err := loadScanConfig(configPath)
	if err != nil {
		return err
	}

	scanReport, fileDiffs, err := runSecretScan(scanConfig, refs)
	if err != nil {
		return fmt.Errorf("failed to run scan: %w", err)
	}

	if scanReport.TotalSecretsFound > 0 {
		err = updateResultsStartAndEndLine(scanReport, fileDiffs)
		if err != nil {
			return err
		}
		removeDuplicateResults(scanReport)
		preReceiveReport := report.PreReceiveReport(scanReport)
		fmt.Print(preReceiveReport)
		err = logReport(logsPath, preReceiveReport)
		if err != nil {
			return err
		}
		os.Exit(1)
	}
	fmt.Print("No secrets detected by Cx Secret Scanner")
	return nil
}

func runSecretScan(scanConfig PreReceiveConfig, refs []string) (*reporting.Report, map[string]*report.FileInfo, error) {
	zerolog.SetGlobalLevel(zerolog.Disabled)

	// Create the scanner.
	scanner := twoms.NewScanner()
	itemsCh := make(chan twoms.ScanItem)
	reportCh := make(chan *reporting.Report)
	errScanCh := make(chan error, 1)

	go func() {
		scanReport, err := scanner.ScanDynamic(itemsCh, twoms.ScanConfig{
			IgnoreResultIds: scanConfig.IgnoreSecret,
			IgnoreRules:     scanConfig.IgnoreRule,
		})
		if err != nil {
			errScanCh <- err
			return
		}
		reportCh <- scanReport
	}()

	fileDiffs, err := runDiffParsing(itemsCh, scanConfig, refs)
	if err != nil {
		return nil, nil, err
	}
	close(itemsCh)

	// Wait for the scanner to finish.
	select {
	case rep := <-reportCh:
		return rep, fileDiffs, nil
	case err = <-errScanCh:
		return nil, nil, err
	}
}

func runDiffParsing(itemsChan chan twoms.ScanItem, config PreReceiveConfig, refs []string) (map[string]*report.FileInfo, error) {
	fileDiffs := make(map[string]*report.FileInfo)
	scanner := bufio.NewScanner(os.Stdin)

	fileExclusion := configExcludesToGitExcludes(config.ExcludePath)

	for _, line := range refs {
		// Expect exactly three fields: oldRev, newRev, and refName.
		parts := strings.Fields(line)
		if len(parts) != 3 {
			return nil, fmt.Errorf("invalid input line: %s", line)
		}
		oldRev, newRev, refName := parts[0], parts[1], parts[2]

		var diffCmd *exec.Cmd
		var args []string
		switch {
		case oldRev == zeroRev && newRev != zeroRev:
			// New ref — show the patch for the root commit.
			args = []string{"log", "-p", "--root", newRev}
		case newRev == zeroRev:
			// Ref deletion — nothing to diff.
			continue
		default:
			// Normal update: diffs between old and new revisions.
			args = []string{"log", "-p", fmt.Sprintf("%s..%s", oldRev, newRev)}
		}

		// Add pathspec separator and exclusions
		args = append(args, "--", ".")
		args = append(args, fileExclusion...)

		diffCmd = exec.Command("git", args...)

		// Get the stdout pipe to parse the log output.
		pipe, err := diffCmd.StdoutPipe()
		if err != nil {
			return nil, fmt.Errorf("failed to get stdout pipe for ref %s: %w", refName, err)
		}
		if err = diffCmd.Start(); err != nil {
			return nil, fmt.Errorf("failed to start log command for ref %s: %w", refName, err)
		}

		diffs, err := gitdiff.Parse(pipe)
		if err != nil {
			return nil, fmt.Errorf("failed to parse diff for ref %s: %w", refName, err)
		}
		for file := range diffs {
			processFileDiff(file, itemsChan, fileDiffs)
		}
		if err = diffCmd.Wait(); err != nil {
			return nil, fmt.Errorf("log command failed for ref %s: %w", refName, err)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed reading input: %w", err)
	}

	return fileDiffs, nil
}

func processFileDiff(file *gitdiff.File, itemsChan chan twoms.ScanItem, fileDiffs map[string]*report.FileInfo) {
	if file.PatchHeader == nil {
		// When parsing the PatchHeader, the token size limit may be exceeded, resulting in a nil value.
		// This scenario is unlikely but may cause the scan to never complete.
		file.PatchHeader = &gitdiff.PatchHeader{}
		file.PatchHeader.SHA = unknownCommit
	}

	log.Debug().Msgf("file: %s; Commit: %s", file.NewName, file.PatchHeader.Title)

	// Skip binary files.
	if file.IsBinary {
		return
	}

	// Extract the changes (added and removed) from the text fragments.
	addedChanges, removedChanges := extractChanges(file.TextFragments)

	var fileName string
	if file.IsDelete {
		fileName = file.OldName
	} else {
		fileName = file.NewName
	}
	id := fmt.Sprintf("hooks-%s", fileName)

	if addedChanges != "" {
		source := fmt.Sprintf("Added:%s:%s", file.PatchHeader.SHA, fileName)
		itemsChan <- twoms.ScanItem{
			Content: &addedChanges,
			ID:      id,
			Source:  source,
		}
		fileDiffs[source] = &report.FileInfo{File: file, ContentType: plugins.AddedContent}
	}

	if removedChanges != "" {
		source := fmt.Sprintf("Deleted:%s:%s", file.PatchHeader.SHA, fileName)
		itemsChan <- twoms.ScanItem{
			Content: &removedChanges,
			ID:      id,
			Source:  source,
		}
		fileDiffs[source] = &report.FileInfo{File: file, ContentType: plugins.RemovedContent}
	}
}

func extractChanges(fragments []*gitdiff.TextFragment) (added string, removed string) {
	var addedBuilder, removedBuilder strings.Builder

	for _, tf := range fragments {
		if tf == nil {
			continue
		}
		for i := range tf.Lines {
			switch tf.Lines[i].Op {
			case gitdiff.OpAdd:
				addedBuilder.WriteString(tf.Lines[i].Line)
			case gitdiff.OpDelete:
				removedBuilder.WriteString(tf.Lines[i].Line)
			default:
			}
			// Clean up the line content to free memory.
			tf.Lines[i].Line = ""
		}
	}

	return addedBuilder.String(), removedBuilder.String()
}

func updateResultsStartAndEndLine(report *reporting.Report, fileDiffs map[string]*report.FileInfo) error {
	for id, secrets := range report.Results {
		for secretIndex, secret := range secrets {
			fileDiff := fileDiffs[secret.Source]
			newStartLine, newEndLine, err := plugins.GetGitStartAndEndLine(&plugins.GitInfo{
				Hunks:       fileDiff.File.TextFragments,
				ContentType: fileDiff.ContentType,
			}, secret.StartLine, secret.EndLine)
			if err != nil {
				return err
			}
			report.Results[id][secretIndex].StartLine = newStartLine
			report.Results[id][secretIndex].EndLine = newEndLine
		}
	}
	return nil
}

func removeDuplicateResults(report *reporting.Report) {
	seenKeys := make(map[string]struct{})
	newResults := make(map[string][]*secrets.Secret, len(report.Results))

	for category, list := range report.Results {
		for _, sec := range list {
			key := fmt.Sprintf(
				"%s|%s|%s|%d|%d|%d|%d",
				sec.ID, sec.Source, sec.RuleID,
				sec.StartLine, sec.EndLine,
				sec.StartColumn, sec.EndColumn,
			)
			if _, found := seenKeys[key]; !found {
				seenKeys[key] = struct{}{}
				newResults[category] = append(newResults[category], sec)
			}
		}
	}

	report.Results = newResults
	report.TotalSecretsFound = len(seenKeys)
}

// skipScan returns true if the special "skip-secrets" push-option is present.
func skipScan() bool {
	// Read how many push-options were sent
	countStr := os.Getenv(gitPushOptionCount)
	if countStr == "" {
		// No push-options supported or none sent
		return false
	}

	count, err := strconv.Atoi(countStr)
	if err != nil || count == 0 {
		// Invalid count or zero options
		return false
	}

	// Inspect each push-option
	for i := 0; i < count; i++ {
		key := gitPushOptionPrefix + strconv.Itoa(i)
		val := os.Getenv(key)
		if val == skipScanKeyword {
			return true
		}
	}

	return false
}

// readRefsFromStdin returns each “oldRev newRev refName” line as a slice.
func readRefsFromStdin() ([]string, error) {
	scanner := bufio.NewScanner(os.Stdin)
	var refs []string
	for scanner.Scan() {
		refs = append(refs, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return refs, nil
}
