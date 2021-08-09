package util

import (
	"bytes"
	"fmt"
	"github.com/MakeNowJust/heredoc"
	"regexp"
	"strings"

	"github.com/gookit/color"
	"github.com/spf13/cobra"
)

func RootHelpFunc(command *cobra.Command) {
	var commands []string

	for _, c := range command.Commands() {
		s := rpad(c.Name()+":", c.NamePadding()) + c.Short
		commands = append(commands, s)
	}

	type helpEntry struct {
		Title string
		Body  string
	}

	longText := command.Long
	if longText == "" {
		longText = command.Short
	}

	var helpEntries []helpEntry
	if longText != "" {
		helpEntries = append(helpEntries, helpEntry{"", longText})
	}
	helpEntries = append(helpEntries, helpEntry{"USAGE", command.UseLine()})

	if len(commands) > 0 {
		helpEntries = append(helpEntries, helpEntry{"COMMANDS", strings.Join(commands, "\n")})
	}

	flagUsages := command.LocalFlags().FlagUsages()
	if flagUsages != "" {
		helpEntries = append(helpEntries, helpEntry{"FLAGS", dedent(flagUsages)})
	}

	inheritedFlagUsages := command.InheritedFlags().FlagUsages()
	if inheritedFlagUsages != "" {
		helpEntries = append(helpEntries, helpEntry{"GLOBAL FLAGS", dedent(inheritedFlagUsages)})
	}
	if command.Example != "" {
		helpEntries = append(helpEntries, helpEntry{"EXAMPLES", command.Example})
	}

	if _, ok := command.Annotations["command:doc"]; ok {
		helpEntries = append(helpEntries, helpEntry{"DOCUMENTATION", command.Annotations["command:doc"]})
	}

	helpEntries = append(helpEntries, helpEntry{"QUICK START GUIDE",
		"https://checkmarx.atlassian.net/wiki/x/mIKctw"})

	if _, ok := command.Annotations["utils:env"]; ok {
		helpEntries = append(helpEntries, helpEntry{"ENVIRONMENT VARIABLES", command.Annotations["utils:env"]})
	}

	helpEntries = append(helpEntries, helpEntry{"LEARN MORE",
		heredoc.Doc(`Use 'cx <command> <subcommand> --help' for more information about a command.
		Read the manual at https://checkmarx.atlassian.net/wiki/x/MYDCkQ`)})

	out := command.OutOrStdout()
	for _, e := range helpEntries {
		if e.Title != "" {
			color.SetOutput(out)
			color.Bold.Println(e.Title)
			fmt.Fprintln(out, indent(strings.Trim(e.Body, "\r\n"), "  "))
		} else {
			// If there is no title print the body as is
			fmt.Fprintln(out, e.Body)
		}
		fmt.Fprintln(out)
	}
}

func indent(s, indent string) string {
	var lineRE = regexp.MustCompile(`(?m)^`)

	if strings.TrimSpace(s) == "" {
		return s
	}
	return lineRE.ReplaceAllLiteralString(s, indent)
}

func rpad(s string, padding int) string {
	template := fmt.Sprintf("%%-%ds ", padding)
	return fmt.Sprintf(template, s)
}

func dedent(s string) string {
	lines := strings.Split(s, "\n")
	minIndent := -1

	for _, l := range lines {
		if l == "" {
			continue
		}

		indent := len(l) - len(strings.TrimLeft(l, " "))
		if minIndent == -1 || indent < minIndent {
			minIndent = indent
		}
	}

	if minIndent <= 0 {
		return s
	}

	var buf bytes.Buffer
	for _, l := range lines {
		fmt.Fprintln(&buf, strings.TrimPrefix(l, strings.Repeat(" ", minIndent)))
	}
	return strings.TrimSuffix(buf.String(), "\n")
}
