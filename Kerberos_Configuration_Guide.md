# Kerberos Authentication - Configuration Guide

## Configuration Parameters

### Required Parameters

| Parameter | Flag | Environment Variable | Required | Description |
|-----------|------|---------------------|----------|-------------|
| **Proxy URL** | `--proxy` | `CX_HTTP_PROXY` | ✅ Yes | Proxy server URL |
| **Auth Type** | `--proxy-auth-type kerberos` | `CX_PROXY_AUTH_TYPE=kerberos` | ✅ Yes | Enable Kerberos |
| **Service Principal** | `--proxy-kerberos-spn` | `CX_PROXY_KERBEROS_SPN` | ✅ Yes | Proxy SPN |

### Optional Parameters

| Parameter | Flag | Environment Variable | Default | Description |
|-----------|------|---------------------|---------|-------------|
| **krb5.conf Path** | `--proxy-kerberos-krb5-conf` | `CX_PROXY_KERBEROS_KRB5_CONF` | `/etc/krb5.conf` (Linux) `C:\Windows\krb5.ini` (Windows) | Kerberos config file |
| **Credential Cache** | `--proxy-kerberos-ccache` | `CX_PROXY_KERBEROS_CCACHE` | `/tmp/krb5cc_$(id -u)` (Linux) Windows Credential Manager (Windows) | Ticket cache location |

### Standard Kerberos Variables

| Variable | Purpose | When to Use |
|----------|---------|-------------|
| `KRB5CCNAME` | Custom credential cache location | Alternative cache path |
| `KRB5_CONFIG` | Custom krb5.conf file location | Alternative config file |

## Configuration Precedence

| Priority | Source | Example |
|----------|--------|---------|
| **1 (Highest)** | Command-Line Flags | `--proxy-kerberos-spn HTTP/proxy.company.com` |
| **2** | Environment Variables | `export CX_PROXY_KERBEROS_SPN=HTTP/proxy.company.com` |
| **3** | Configuration File | `cx_proxy_kerberos_spn: HTTP/proxy.company.com` |
| **4 (Lowest)** | Default Values | Platform defaults |

## Examples

### Example 1: Project List with Flags

```bash
# Get Kerberos tickets first
kinit user@COMPANY.COM

# Run command with flags
cx project list \
  --proxy "http://proxy.company.com:8080" \
  --proxy-auth-type kerberos \
  --proxy-kerberos-spn "HTTP/proxy.company.com"
```

### Example 2: Scan with Environment Variables

```bash
# Get Kerberos tickets
kinit user@COMPANY.COM

# Set environment variables
export CX_HTTP_PROXY="http://proxy.company.com:8080"
export CX_PROXY_AUTH_TYPE=kerberos
export CX_PROXY_KERBEROS_SPN="HTTP/proxy.company.com"

# Run scan (environment variables applied automatically)
cx scan create \
  --project-name "projectName" \
  -s "codePath" \
  --branch "branchName"
```

### Example 3: Mixed Configuration

```bash
# Get tickets
kinit user@COMPANY.COM

# Set base configuration via environment
export CX_HTTP_PROXY="http://proxy.company.com:8080"
export CX_PROXY_AUTH_TYPE=kerberos

# Override SPN with flag (higher precedence)
cx project list --proxy-kerberos-spn "HTTP/specific-proxy.company.com"
```

## Setup Steps

1. **Get Kerberos tickets**: `kinit user@COMPANY.COM`
2. **Verify tickets**: `klist`
3. **Configure proxy settings** (flags or environment variables)
4. **Run AST CLI commands**

## Quick Reference

| Action | Command |
|--------|---------|
| **Get tickets** | `kinit user@COMPANY.COM` |
| **Check tickets** | `klist` |
| **Renew tickets** | `kinit -R` |
| **Test connectivity** | `curl --proxy "http://proxy.company.com:8080" http://httpbin.org/ip` |
