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

This document provides many examples of using the AST CLI but it's impossible to cover every possible action. You can  you can always fall back to the (--help or -h) option, ex:

### Windows

``` powershell
cx.exe --help
```

### Linux/Mac
``` bash
./cx -h
```

You will notice help shows a list of available commands and a summary of global parameters. The (--help) command also lets you dig into commands for more context specific help, ex:

``` bash
# Show help for the scan command
./cx scan -h
Manage scans

Usage:
  cx scan [command]

Available Commands:
  cancel      Cancel one or more scans from running
  create      Create and run a new scan
  delete      Deletes one or more scans
  list        List all scans in the system
  show        Show information about a scan
  tags        Get a list of all available tags to filter by
  workflow    Show information about a scan workflow
  
# At this point you can dig into the "create" command
./cx scan create -h
Create and run a new scan

Usage:
  cx scan create [flags]

Flags:
  -d, --directory string             A path to directory with sources to scan
  -f, --filter string                Source file filtering pattern
      --preset-name string           The name of the Checkmarx preset to use.
      --project-name string       
      ....
```

You may have noticed the parameters accepted by the CLI vary based on the commands issued but the following parameters are available throughout the CLI command hierarchy:

- (--base-uri), the URL of the AST server.
- (--base-auth-uri), optionally provides alternative KeyCloak endpoint to (--base-uri).
- (--client-id), the client ID used for authentication (see Authentication documentation).
- (--secret), the secret that corrosponds to the client-id  (see Authentication documentation).
- (--token), the token to authenticate with (see Authentication documentation).
- (--proxy), optional proxy server to use (see Proxy Support documentation).
- (--insecure), indicates CLI should ignore TLS certificate validations.
- (--profile), specifies the CLI profile to store options in (see Environment and Configuration documentation).

Many CLI variables can be provided using environment variables, configuration variables or CLI parameters. The follow precidence is used when the same value is found in settings:

1. CLI parameters, these always overide configuration and environment variables.
2. Configuration variables always overide environment variables.
3. Environment variables are the first order precidence. 

## CLI Configurations

The CLI allows you to permanently store some CLI options in configuration files. The configuration files are kept in the users home directory under a subdirectory named  ($HOME/.checkmarx). 

``` bash
./cx configure set cx_base_uri "http://<your-server>[:<port>]"
./cx configure set cx_ast_access_key_id <your-key>
./cx configure set cx_ast_access_key_secret <your-secret>
./cx configure set cx_http_proxy <your-proxy>
./cx configure set cx_token <your-token>
```

The (--profile) option provides a powerful tool to quickly switch through differnent sets of configurations. You can add a profile name to any CLI command and it utilize the corsponding profile settings. The following example setups up altnerative profile named "test" and calls a CLI command utilizing it.

``` bash
# Create the alternative profile
./cx configure set cx_base_uri "http://your-test-server" --profile test
# Use the cx_base_uri from the "test" profile.
./cx scan list --profile test
# This uses the default profile (if it exists)
./cx scan list
```

The configure command supports an interactive mode that prompt you for the following common options: base-uri, client ID and secret, ex:

``` bash
./cx configure
AST Base URI [http://<your-domain]: <your-updated-domain>
AST Access Key [******f23d]: <your-updated-key>
AST Key Secret [******8913]: <your-updated-secret>
```

If the CLI has previously stored values they will show up like you see in the previous example, you can just press enter if you want to keep the existing value.

These values can be stored in CLI configurations:

- cx_token: the token to authenticate with (see Authentication documentation).
- cx_base-uri: the URL of the AST server.
- cx_http_proxy: optional proxy server to use (see Proxy Support documentation). 
- cx_ast_access_key_id: the client ID used for authentication (see Authentication documentation).
- cx_ast_access_key_secret: the secret that corrosponds to the client-id  (see Authentication documentation).

## Authentication

The CLI supports token and key/secret based authentication.

Token based authentication is the easiest method to use in CI/CD environments. Tokens are generated through KeyCloak and can be created with a predictable lifetime. Once you have a token you can use it from the CLI like this:

``` bash
./cx --token <your-token> scan list 
```

You can optionally configure the token into your stored CLI configuration values like this:

``` bash
./cx configure set cx_token <your-token>
# The following command will automatically use the stored token
./cx scan list
```

You can also store the token in the environment like this:

``` bash
export CX_TOKEN=<your-token>
./cx scan list
```

Key/secret authentication requires you to first use an AST username and password to create the key and secret for authentication. The following example shows how to create a key/secret and then use it:

``` bash
./cx auth register -u <username> -p <password>
CX_AST_ACCESS_KEY_ID=<generated-key>
CX_AST_ACCESS_KEY_SECRET=<generated-secret>
```

Once you generated your key and secret they can be used like this:

``` bash
./cx --client-id <your-key> --secret <your-secret> scan list 
```

You can optionally configure the key/secret into your stored CLI configuration values like this:

``` bash
./cx configure set cx_ast_access_key_id <your-key>
./cx configure set cx_ast_access_key_secret <your-secret>
# The following command will automatically use the stored key/secret
./cx scan list
```

You can also store the key/secret in the environment like this:

``` bash
export CX_AST_ACCESS_KEY_ID=<your-key>
export CX_AST_ACCESS_KEY_SECRET=<your-secret>
./cx scan list
```



## Triggering Scans

You need to specify a project using the (--project-name) parameter when you create a scan. If the project doesn't exist then it will be created automatically; however, if the the project exists it will be reused. The following examples will all use the same project name for simplicity.

You can optionally specify the name of the preset to use when scanning projects using the (--preset-name) parameter. If you don't specify the preset name then "Checkmarx Default" will be used. 

You can indicate if an incremental or full scan should be performed with the (--incremental) parameter. If you don't provide the incremental flag then a full scan will be triggered.

The (--project-type) parameter is used to indicate which types of scan should be performed by AST. You can provide a comma separated list of scan types if you want multiple scans to be performed. If you ommit this paramteter only a  SAST scan will be performed.

You have three options when it comes to creating scans, the most important thing you need to decide is where the scan is going to come from:

1. A zip file with your source code.
2. A directory with your source code.
3. A host git repo.

**NOTE**: for simplicity the following examples assume you have stored your authentication and base-uri information in either environment variables or CLI configuration parameters. These values are required but will not appear in the commands.

**NOTE**: to show different real world situations optional parameters will sometimes but not always be used.

After you create a scan the CLI will wait until it is completed or an error has been encountered. The default polling interval is 5 seconds but you can overide that with the (--wait-delay) option. You can also turn off the wait mode with (--nowait true).

Scanning zipped code archives can be achieved like this:

``` bash
./cx scan create -s <your-file>.zip --project-name "testproj" --preset-name "Checkmarx Default" --incremental "false" --project-type "sast" -f <your-source>.zip
```

If you decide to scan a local directory you can provide filters that determine which resources are sent to the AST server. The filters are based on an inclusion and exclusion model. The following example shows how to scan a folder:

``` bash
./cx scan create -d <path-to-your-folder> -f "s*.go" --project-name "testproj" --incremental "false" --project-type "sast" 
```

The filter in this case will include any go files that start with an 's'. You can include more then one set of files and directories by separating the inclusion patterns with a comma, example:

``` bash
./cx scan create -d <path-to-your-folder> -f "s*,*.txt" --project-name "testproj" --preset-name "Checkmarx Default" --incremental "false"
```

In this previous example any files that start with 's' will be included, as well as any files that end with '.txt'. You can add an exclusion into the list by prepending the pattern with a '!'. The following query demonstrates exclusion by filtering files that end with 'zip':

``` bash
./cx scan create -d <path-to-your-folder> -f "s*,*.txt,!*.zip" --project-name "testproj" --preset-name "Checkmarx Default" --incremental "false" --project-type "sast" 
```

Git repositories can be scanned like this:

``` bash
./cx scan create -r <your-repo-url> --project-name "testproj" 
```

When you're scanning repos AST will fetch the code directly from the repository.

You can disable polling mode like this:

``` bash
./cx scan create -r <your-repo-url> --project-name "testproj" --nowait true
```


## Managing Projects

You can create, delete, list or show details about AST projects using the CLI. You specifically create projects before trigging scans though, the (scan create) will automatically create projects that don't exist for you. The commands just provide a help way to work with projects.

You can create projects like this:

``` bash
./cx project create --branch "test" --project-name "createTest" --repo-url "https://github.com/tsunez/checkmarxTest.git"

Project ID       Name       Created at          Updated at          Tags Groups 
----------       ----       ----------          ----------          ---- ------ 
56939423....    createTest  03-22-21 08:27:34   03-22-21 08:27:34   []   [] 
```

The only required parameter is (--project-name). If the (--repo-url) or (--branch) don't make sense for your purposes then just skip them.

You can list existing projects like this:

``` bash
./cx project list

Project ID       Name       Created at          Updated at          Tags Groups 
----------       ----       ----------          ----------          ---- ------ 
56939423....    createTest  03-22-21 08:27:34   03-22-21 08:27:34   []   [] 
```

You can show the details about a specific project like this:

``` bash
./cx project list <your-project-id>

Project ID       Name       Created at          Updated at          Tags Groups 
----------       ----       ----------          ----------          ---- ------ 
56939423....    createTest  03-22-21 08:27:34   03-22-21 08:27:34   []   [] 
```

Finally you can delete a project like this:

``` bash
./cx project delete <your-project-id>
```



## Retreiving Results

...todo, we're still waiting to work out result retreival

## Proxy Support

The CLI full supports proxy servers and optional proxy server authentication. When the proxy server variable is found all CLI operations will be routed through the target server. Proxy server URLs should like this: "http[s]://your-server.com:[port]"

Proxy support is enabled by creating an environment variable named CX_HTTP_PROXY. You can also specify the proxy by storing the proxy URL in a CLI configuration variable (cx_proxy_http), or directly through with the CLI with the parameter  (--proxy).

The following example demonstraights the use of a proxy server:

``` bash
./cx scan list --proxy "http://<your-proxy>:8081"
```



## Environment Variables

| Environment Variable         | Description                                                  |
| ---------------------------- | ------------------------------------------------------------ |
| **CX_AST_ACCESS_KEY_ID**     | Key portion of key/secret authentication pair.               |
| **CX_AST_ACCESS_KEY_SECRET** | Secret portion of key/secret authentication pair.            |
| **CX_TOKEN**                 | Token for token based authentication.                        |
| **CX_BASE_URI**              | The URI of the AST server.                                   |
| **CX_BASE_IAM_URI**          | The URI of KeyCloak instance. This optional and only required when you're not using AST's built in KeyCloak instance. |
| **CX_HTTP_PROXY**            | When provided this variable will trigger the CLI to use the proxy server pointed to (see proxy support documentation). |


