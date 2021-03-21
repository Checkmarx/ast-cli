[![CircleCI](https://circleci.com/gh/CheckmarxDev/ast-cli/tree/master.svg?style=svg&circle-token=32eeef7505db60c11294e63db64e70882bde83b0)](https://circleci.com/gh/CheckmarxDev/ast-cli/tree/master)
## Building from source code
### Windows 
``` powershell
setx GOOS=windows 
setx GOARCH=am
go build -o ./bin/cx.exe ./cmd
```

### Linux

``` bash
export GOARCH=amd64
export GOOS=linux
go build -o ./bin/cx ./cmd
```

### Macintosh

``` bash
export GOOS=darwin 
export GOARCH=amd64
go build -o ./bin/cx-mac ./cmd
```

** **

## Basic CLI Operation

### Windows
``` powershell
cx.exe [commands]
```

### Linux/Mac
``` bash
./cx [commands]
```

The parameters accepted by the CLI vary based on the commands issued and they will be described thoroughly throughout this document. The following global parameters affect all actions:

- (--base-uri), the URL of the AST server.
- (--base-auth-uri), optionally provides alternative KeyCloak endpoint to (--base-uri).
- (--client-id), the client ID used for authentication (see Authentication documentation).
- (--secret), the secret that corrosponds to the client-id  (see Authentication documentation).
- (--token), the token to authenticate with (see Authentication documentation).
- (--proxy), optional proxy server to use (see Proxy Support documentation).
- (--insecure), indicates CLI should ignore TLS certificate validations.
- (--profile), specifies the CLI profile to store options in (see Environment and Configuration documentation).

## CLI Configurations

todo...

The CLI allows you to permanently store some CLI options in configuration files. The configuration files are kept in the users home directory under a subdirectory called (.checkmarx). It's also possible to maintain more then one configuration using the (--profile) option. The (--profile) option provides a powerful tool to quickly switch through differnent sets of configurations. 

``` bash
./cx configuration set cx_base_uri "http://your-server:8081"

... more examples
```



## Authentication

The CLI supports token and key/secret based authentication.

Token based authentication is probably the easiest method to use in CI/CD environments. Tokens are generated through KeyCloak and can be created with a predictable lifetime. Once you have a token you can use it from the CLI like this:

``` bash
cx --token <your-token> scan list 
```

You can optionally configure the token into your stored CLI configuration values like this:

``` bash
cx configure set token <your-token>
# The following command will automatically use the stored token
cx scan list
```

You can also store the token in the environment like this:

``` bash
export CX_TOKEN=<your-token>
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

## Triggering Scans



## Retreiving Results



## Managing Projects



## Proxy Support

Proxy support is enabled by creating an environmen variable named _HTTP_PROXY_ or setting the configuration option <fill-in>.

The environment values may be either a complete URL or a "host[:port]", in which case the "http" scheme is assumed. 

## Environment Variables



