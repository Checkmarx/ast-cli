//go:build !integration

package commands

import (
	"testing"
)

func TestDetectShell(t *testing.T) {
	cases := []struct {
		name     string
		psModule string // value for PSModulePath env
		shell    string // value for SHELL env
		want     string
	}{
		{name: "PSModulePath set → powershell", psModule: "C:\\Program Files\\PowerShell\\Modules", shell: "", want: "powershell"},
		{name: "PSModulePath wins over SHELL", psModule: "C:\\Program Files\\PowerShell\\Modules", shell: "/usr/bin/bash", want: "powershell"},
		{name: "bash via SHELL", psModule: "", shell: "/usr/bin/bash", want: "bash"},
		{name: "zsh via SHELL", psModule: "", shell: "/usr/bin/zsh", want: "bash"},
		{name: "fish via SHELL", psModule: "", shell: "/usr/local/bin/fish", want: "fish"},
		{name: "fish wins over bash substring matching", psModule: "", shell: "/usr/local/bin/fishtank", want: "fish"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("PSModulePath", tc.psModule)
			t.Setenv("SHELL", tc.shell)
			got := detectShell()
			if got != tc.want {
				t.Errorf("detectShell() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestFormatEnvAssignment(t *testing.T) {
	cases := []struct {
		name  string
		shell string
		key   string
		value string
		want  string
	}{
		{name: "powershell with token", shell: "powershell", key: "CX_APIKEY", value: "abc.def", want: `$env:CX_APIKEY = "abc.def"`},
		{name: "powershell with empty value clears", shell: "powershell", key: "CX_APIKEY", value: "", want: `$env:CX_APIKEY = ""`},
		{name: "bash with token", shell: "bash", key: "CX_APIKEY", value: "abc.def", want: `export CX_APIKEY="abc.def"`},
		{name: "bash with empty value clears", shell: "bash", key: "CX_APIKEY", value: "", want: `export CX_APIKEY=""`},
		{name: "fish with token", shell: "fish", key: "CX_APIKEY", value: "abc.def", want: `set -gx CX_APIKEY "abc.def"`},
		{name: "unknown shell falls back to bash syntax", shell: "made-up-shell", key: "CX_APIKEY", value: "abc.def", want: `export CX_APIKEY="abc.def"`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := formatEnvAssignment(tc.shell, tc.key, tc.value)
			if got != tc.want {
				t.Errorf("formatEnvAssignment(%q, %q, %q) = %q, want %q", tc.shell, tc.key, tc.value, got, tc.want)
			}
		})
	}
}
