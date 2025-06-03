//go:build integration

package integration

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPreReceive_PushSecrets(t *testing.T) {
	workDir, cleanUp := setUpPreReceiveHookDir(t)
	defer cleanUp()
	assert.NoError(t, os.Chdir(workDir))
	setGlobalGitAccount(t, workDir)

	secretFile := filepath.Join(workDir, "secret.txt")
	err := os.WriteFile(secretFile, []byte("ghp_DDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDD"), 0644)
	assert.NoError(t, err)
	// Git add
	outputCmd := exec.Command("git", "add", "secret.txt")

	// making it workingDir
	outputCmd.Dir = workDir

	output, err := outputCmd.CombinedOutput()
	assert.NoError(t, err, "failed to add files in staging :%s", string(output))

	// Add commit
	commitCmd := exec.Command("git", "commit", "-m", "added secrets file")
	commitCmd.Dir = workDir
	output, err = commitCmd.CombinedOutput()
	assert.NoError(t, err, "Filed to commit :%s", string(output))
	//Pushing

	cmdPush := exec.Command("git", "push")
	cmdPush.Dir = workDir
	output, err = cmdPush.CombinedOutput()
	outputString := string(output)
	assert.Error(t, err, "Failed to push: %s", outputString)
	assert.Contains(t, outputString, "[remote rejected]")
	assert.Contains(t, outputString, "(pre-receive hook declined)")
}

func TestPreReceive_PushWithoutSecrets(t *testing.T) {
	workDir, cleanUp := setUpPreReceiveHookDir(t)
	defer cleanUp()
	assert.NoError(t, os.Chdir(workDir))
	setGlobalGitAccount(t, workDir)

	noSecretFile := filepath.Join(workDir, "without_secret.txt")
	err := os.WriteFile(noSecretFile, []byte("Hello World"), 0644)
	assert.NoError(t, err)
	// Git add
	outputCmd := exec.Command("git", "add", "without_secret.txt")

	// making it workingDir
	outputCmd.Dir = workDir

	output, err := outputCmd.CombinedOutput()
	assert.NoError(t, err, "failed to add files in staging :%s", string(output))

	// Add commit
	commitCmd := exec.Command("git", "commit", "-m", "added without secrets file")
	commitCmd.Dir = workDir
	output, err = commitCmd.CombinedOutput()
	assert.NoError(t, err, "Filed to commit :%s", string(output))
	//Pushing

	cmdPush := exec.Command("git", "push")
	cmdPush.Dir = workDir
	output, err = cmdPush.CombinedOutput()
	outputString := string(output)
	assert.NotContains(t, outputString, "[remote rejected]")
	assert.NotContains(t, outputString, "(pre-receive hook declined)")
}

func TestPreReceive_PushSecrets_and_NoSecretsFile(t *testing.T) {
	workDir, cleanUp := setUpPreReceiveHookDir(t)
	defer cleanUp()
	assert.NoError(t, os.Chdir(workDir))
	setGlobalGitAccount(t, workDir)
	//create non-secret file
	nonSecretFile := filepath.Join(workDir, "without_secret.txt")
	err := os.WriteFile(nonSecretFile, []byte("Hello World"), 0644)
	assert.NoError(t, err)
	//create a secret file
	secretFile := filepath.Join(workDir, "secret1.txt")
	err = os.WriteFile(secretFile, []byte("ghp_DDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDD"), 0644)
	assert.NoError(t, err)
	// Git add
	outputCmd := exec.Command("git", "add", "without_secret.txt", "secret1.txt")
	// making it workingDir
	outputCmd.Dir = workDir

	output, err := outputCmd.CombinedOutput()
	assert.NoError(t, err, "failed to add files in staging :%s", string(output))

	// Add commit
	commitCmd := exec.Command("git", "commit", "-m", "added without secrets file")
	commitCmd.Dir = workDir
	output, err = commitCmd.CombinedOutput()
	assert.NoError(t, err, "Filed to commit :%s", string(output))
	//Pushing
	cmdPush := exec.Command("git", "push")
	cmdPush.Dir = workDir
	output, err = cmdPush.CombinedOutput()
	outputString := string(output)
	assert.Contains(t, outputString, "[remote rejected]")
	assert.Contains(t, outputString, "(pre-receive hook declined)")
	assert.Contains(t, outputString, "Detected 1 secret across 1 commit")
}

func setGlobalGitAccount(t *testing.T, repoName string) {
	// Set global git config
	githubUserName := os.Getenv("GITHUB_USER")
	githubUserEmail := os.Getenv("GITHUB_EMAIL")
	err := exec.Command("git", "-C", repoName, "config", "user.email", githubUserEmail).Run()
	err = exec.Command("git", "-C", repoName, "config", "user.name", githubUserName).Run()
	assert.NoError(t, err)
}

func setUpPreReceiveHookDir(t *testing.T) (workdir string, cleanup func()) {
	orgWorkDir, err := os.Getwd()
	assert.NoError(t, err)
	tempDir := t.TempDir()
	fmt.Println("the current dir" + orgWorkDir)

	//Init a bare repo

	err = exec.Command("git", "init", "--bare", filepath.Join(tempDir, "server")).Run()
	assert.NoError(t, err)
	cxPath := filepath.Join(orgWorkDir, "..", "..", "bin", "cx")
	fmt.Println("the current dir" + cxPath)

	preReceivePath := filepath.Join(tempDir, "server", "hooks", "pre-receive")
	script := fmt.Sprintf(`#!/bin/bash
							  "%s" hooks pre-receive secrets-scan`, cxPath)
	err = os.WriteFile(preReceivePath, []byte(script), 0755)
	assert.NoError(t, err)

	err = exec.Command("git", "clone", filepath.Join(tempDir, "server"), filepath.Join(tempDir, "client")).Run()

	mainDir := filepath.Join(tempDir, "client")
	err = os.Chdir(mainDir)
	assert.NoError(t, err)
	cleanUp := func() {
		assert.NoError(t, os.Chdir(orgWorkDir))
	}

	return mainDir, cleanUp
}
