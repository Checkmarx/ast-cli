[![CircleCI](https://circleci.com/gh/CheckmarxDev/ast-cli/tree/master.svg?style=svg&circle-token=32eeef7505db60c11294e63db64e70882bde83b0)](https://circleci.com/gh/CheckmarxDev/ast-cli/tree/master)
# ast-cli
A CLI project wrapping the AST APIs

## Building from source code
### Windows 
When building an executable for Windows and providing a name,  be sure to explicitly specify the .exe suffix when setting the executable’s name.  
Inside the command prompt run:
**env GOOS=windows GOARCH=amd64 go build -o ./bin/ast.exe ./cmd** 

### Linux
**sudo env GOARCH=amd64 go build -o ./ast ./cmd** 

## Running the CLI

### Windows
"bin/ast.exe" [commands]

### Linux
./ast [commands]

## Authentication

The easiest way is to register you client using:

./ast auth register -u {ADMIN_USER_NAME} -p {ADMIN_USER_PASSWORD} --base-uri http://<REMOTE_IP> 

It wiil Register new oath2 client and outputs its generated credentials:  
**AST_ACCESS_KEY_ID**={The access key ID}  
**AST_ACCESS_KEY_SECRET**={The access key secret}

On Linux just wrap this command with eval e.g: "eval $(ast auth register -u <username> -p <password>)".
On Windows use the SET command with the outputs credentials

You can use [SETX](https://docs.microsoft.com/en-us/windows-server/administration/windows-commands/setx) (windows), [SETENV](https://www.computerhope.com/unix/usetenv.htm)  (linux) for permanent save     

Both access key ID and access key secret can be overridden by the flags **--key** and **--secret** respectively

## Running on remote machine
./ast --base-uri http://{REMOTE_IP} [commands]

## HTTP Proxy Support
Use the environment variables _HTTP_PROXY_, _HTTPS_PROXY_ and _NO_PROXY_ (or the lowercase versions). 
HTTPS_PROXY takes precedence over HTTP_PROXY for https requests.
The environment values may be either a complete URL or a "host[:port]", in which case the "http" scheme is assumed. 
An error is returned if the value is a different form.
