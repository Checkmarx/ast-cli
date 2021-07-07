// +build !integration

package commands

/**
Remove test not supported by AST

func TestQueryHelp(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "help", "query")
	assert.NilError(t, err)
}

func TestQueryNoSub(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "utils", "query")
	assert.NilError(t, err)
}

func TestRunUploadCommandWithFile(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "utils", "query", "upload", "./payloads/nonsense.json")
	assert.NilError(t, err)
	err = executeTestCommand(cmd, "-v", "utils", "query", "upload", "./payloads/uploads.json", "--name", "mock")
	assert.NilError(t, err)
}

func TestRunUploadCommandWithNoRepo(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "utils", "query", "upload")
	assert.Assert(t, err != nil)
}

func TestRunUploadCommandWithActivateFlag(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "utils", "query", "upload", "./payloads/uploads.json", "-a")
	assert.NilError(t, err)
}

func TestRunDownloadCommand(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "utils", "query", "download", "mock")
	assert.NilError(t, err)
	wd, _ := os.Getwd()
	mockFilePath := filepath.Join(wd, QueriesRepoDestFileName)
	mockRepoFile, err := os.Open(mockFilePath)
	assert.NilError(t, err, "failed to open repo mock file")
	defer os.Remove(mockFilePath)
	defer mockRepoFile.Close()
	bytes, err := ioutil.ReadAll(mockRepoFile)
	assert.NilError(t, err, "failed to read repo mock file")
	assert.Assert(t, string(bytes) == wrappers.MockContent)
}

func TestRunListCommand(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "utils", "query", "list")
	assert.NilError(t, err)
}

func TestRunActivateCommandBlank(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "utils", "query", "activate")
	assert.Assert(t, err != nil)
}

func TestRunActivateCommand(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "utils", "query", "activate", "mock")
	assert.NilError(t, err)
}

func TestRunDeleteCommandBlank(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "utils", "query", "delete")
	assert.Assert(t, err != nil)
}

func TestRunDeleteCommand(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "-v", "utils", "query", "delete", "mock")
	assert.NilError(t, err)
}
*/
