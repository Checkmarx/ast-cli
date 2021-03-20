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
"bin/cx.exe" [commands]

### Linux/Mac
./cx [commands]

The parameters accepted by the CLI vary based on the command issued but the following parameters effect all actions:

- (--base-uri), the URL of the AST server.

- (--base-auth-uri)

- (--client-id), the client ID used for authentication (see Authentication documentation).

- (--secret), the secret that corrosponds to the client-id  (see Authentication documentation).

- (--token), the token to authenticate with (see Authentication documentation).

- (--proxy), optional proxy server to use (see Proxy Support documentation).

- (--insecure)

- (--profile)

  





## Authentication

The CLI supports token and key/secret based authentication.

Token based authentication is probably the easiest authentication method to use in CI/CD environments. Tokens are generated through KeyCloak and can be created with a predictable lifetime. Once you have a token you can use it from the CLI like this:

``` bash
cx --token <your-token> scan list 
```

You can also configure the token into your stored CLI configuration values like this:

``` bash
cx configure set token <your-token>
# The following command will automatically use the stored component
cx scan list
```

The key/secret authentication requires you to use an AST username and password to create the key and secret for authentication. The following example shows how to create a key/secret and then use it:

``` bash
cx auth register -u <username> -p <password>
CX_AST_ACCESS_KEY_ID=<generated-key>
CX_AST_ACCESS_KEY_SECRET=<generated-secret>

```





The easiest way is to register you client using:

./ast auth register -u {ADMIN_USER_NAME} -p {ADMIN_USER_PASSWORD} --base-uri http://<REMOTE_IP> 

It wiil Register new oath2 client and outputs its generated credentials:  
**AST_ACCESS_KEY_ID**={The access key ID}  
**AST_ACCESS_KEY_SECRET**={The access key secret}

On Linux just wrap this command with eval e.g: "eval $(ast auth register -u <username> -p <password>)".
On Windows use the SET command with the outputs credentials

You can use [SETX](https://docs.microsoft.com/en-us/windows-server/administration/windows-commands/setx) (windows), [SETENV](https://www.computerhope.com/unix/usetenv.htm)  (linux) for permanent save     

Both access key ID and access key secret can be overridden by the flags **--key** and **--secret** respectively

## Proxy Support

Proxy support is enabled by creating an environmen variable named _HTTP_PROXY_ or setting the configuration option <fill-in>.

The environment values may be either a complete URL or a "host[:port]", in which case the "http" scheme is assumed. 

