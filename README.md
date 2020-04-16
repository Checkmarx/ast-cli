[![CircleCI](https://circleci.com/gh/CheckmarxDev/ast-cli/tree/master.svg?style=svg&circle-token=32eeef7505db60c11294e63db64e70882bde83b0)](https://circleci.com/gh/CheckmarxDev/ast-cli/tree/master)
# ast-cli
A CLI project wrapping the AST APIs

## Building from source code
### Windows 
When building an executable for Windows and providing a name,  be sure to explicitly specify the .exe suffix when setting the executable’s name.  
Inside the command prompt run:
**env GOOS=windows GOARCH=amd64 go build -o ./bin/ast.exe ./cmd** 

## Running the CLI

### Windows
"bin/ast.exe" [commands]

## Authentication
In order to authenticate with AST there are 3 environment variables that needs to be set:
**AST_AUTHENTICATION_URI**: The authentication URI used by AST
**AST_ACCESS_KEY_ID**: The access key ID
**AST_ACCESS_KEY_SECRET**: The access key secret

Both access key ID and access key secret can be overriden by the flags **--key** and **--secret** respectively
