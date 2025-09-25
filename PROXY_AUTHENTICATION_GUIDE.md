# Proxy Authentication Guide

This guide explains how to configure proxy authentication using NTLM and Kerberos protocols with the AST CLI tool.

## Overview

The AST CLI supports three types of proxy authentication:
- **Basic**: Standard username/password authentication
- **NTLM**: Windows NT LAN Manager authentication
- **Kerberos**: MIT Kerberos authentication protocol

## Configuration Methods

You can configure proxy authentication using either **command-line flags** or **environment variables**.

---

## 🔧 Basic Proxy Authentication

### Command Line Flags
```bash
cx scan create --proxy http://username:password@proxy.company.com:8080
```

### Environment Variables
```bash
export HTTP_PROXY=http://username:password@proxy.company.com:8080
# or
export CX_HTTP_PROXY=http://username:password@proxy.company.com:8080
```

---

## 🔐 NTLM Proxy Authentication

### When to Use NTLM
- Corporate environments using Windows-based proxy servers
- Active Directory domain authentication required
- Windows NTLM challenge-response authentication

### Command Line Flags
```bash
cx scan create \
  --proxy http://proxy.company.com:8080 \
  --proxy-auth-type ntlm \
  --proxy-ntlm-domain COMPANY_DOMAIN
```

### Environment Variables
```bash
export CX_HTTP_PROXY=http://username:password@proxy.company.com:8080
export CX_PROXY_AUTH_TYPE=ntlm
export CX_PROXY_NTLM_DOMAIN=COMPANY_DOMAIN
```

### NTLM Configuration Details

| Flag | Environment Variable | Description | Required |
|------|---------------------|-------------|----------|
| `--proxy` | `CX_HTTP_PROXY` | Proxy URL with credentials | ✅ Yes |
| `--proxy-auth-type ntlm` | `CX_PROXY_AUTH_TYPE=ntlm` | Enable NTLM authentication | ✅ Yes |
| `--proxy-ntlm-domain` | `CX_PROXY_NTLM_DOMAIN` | Windows domain name | ✅ Yes |

### NTLM Example
```bash
# Full NTLM configuration
cx scan create \
  --proxy http://john.doe:mypassword@proxy.company.com:8080 \
  --proxy-auth-type ntlm \
  --proxy-ntlm-domain COMPANY \
  --source-dir /path/to/source
```

---

## 🎫 Kerberos Proxy Authentication

### Prerequisites
1. **Kerberos tickets**: Obtain valid Kerberos tickets using `kinit`
2. **SPN configuration**: Know the Service Principal Name for your proxy
3. **krb5.conf**: Proper Kerberos configuration file

### Command Line Flags
```bash
cx scan create \
  --proxy http://proxy.company.com:8080 \
  --proxy-auth-type kerberos \
  --proxy-kerberos-spn HTTP/proxy.company.com
```

### Environment Variables
```bash
export CX_HTTP_PROXY=http://proxy.company.com:8080
export CX_PROXY_AUTH_TYPE=kerberos
export CX_PROXY_KERBEROS_SPN=HTTP/proxy.company.com
```

### Kerberos Configuration Details

| Flag | Environment Variable | Description | Required |
|------|---------------------|-------------|----------|
| `--proxy` | `CX_HTTP_PROXY` | Proxy URL (no credentials needed) | ✅ Yes |
| `--proxy-auth-type kerberos` | `CX_PROXY_AUTH_TYPE=kerberos` | Enable Kerberos authentication | ✅ Yes |
| `--proxy-kerberos-spn` | `CX_PROXY_KERBEROS_SPN` | Service Principal Name for proxy | ✅ Yes |
| `--proxy-kerberos-krb5-conf` | `CX_PROXY_KERBEROS_KRB5_CONF` | Path to krb5.conf file | ❌ Optional |
| `--proxy-kerberos-ccache` | `CX_PROXY_KERBEROS_CCACHE` | Path to credential cache | ❌ Optional |

### Kerberos Setup Steps

#### 1. Obtain Kerberos Tickets
```bash
# Get Kerberos tickets for your user
kinit username@REALM.COM

# Verify tickets are available
klist
```

#### 2. Configure SPN
```bash
# Example SPN format for HTTP proxy
--proxy-kerberos-spn HTTP/proxy.company.com

# Example SPN for specific port
--proxy-kerberos-spn HTTP/proxy.company.com:8080
```

#### 3. Run with Kerberos Authentication
```bash
cx scan create \
  --proxy http://proxy.company.com:8080 \
  --proxy-auth-type kerberos \
  --proxy-kerberos-spn HTTP/proxy.company.com \
  --source-dir /path/to/source
```

### Advanced Kerberos Configuration

#### Custom krb5.conf Location
```bash
cx scan create \
  --proxy http://proxy.company.com:8080 \
  --proxy-auth-type kerberos \
  --proxy-kerberos-spn HTTP/proxy.company.com \
  --proxy-kerberos-krb5-conf /etc/custom/krb5.conf
```

#### Custom Credential Cache
```bash
cx scan create \
  --proxy http://proxy.company.com:8080 \
  --proxy-auth-type kerberos \
  --proxy-kerberos-spn HTTP/proxy.company.com \
  --proxy-kerberos-ccache /tmp/custom_krb5cc
```

---

## 🌍 Default Locations

### Kerberos Configuration Files

**Linux/macOS:**
- krb5.conf: `/etc/krb5.conf`
- Credential cache: `/tmp/krb5cc_$(id -u)`

**Windows:**
- krb5.conf: `C:\Windows\krb5.ini`
- Credential cache: Managed by Windows credential manager

### Environment Variables for Kerberos
```bash
# Override default credential cache location
export KRB5CCNAME=/path/to/custom/ccache

# Standard Kerberos environment variable
export KRB5_CONFIG=/path/to/custom/krb5.conf
```

---

## 📋 Complete Examples

### NTLM Corporate Environment
```bash
#!/bin/bash
# NTLM proxy authentication example

export CX_HTTP_PROXY=http://jdoe:password123@proxy.corp.com:8080
export CX_PROXY_AUTH_TYPE=ntlm
export CX_PROXY_NTLM_DOMAIN=CORP

cx scan create \
  --project-name "MyProject" \
  --source-dir /workspace/myapp \
  --branch main
```

### Kerberos Enterprise Environment
```bash
#!/bin/bash
# Kerberos proxy authentication example

# 1. Get Kerberos tickets
kinit jdoe@CORP.COM

# 2. Configure proxy with Kerberos
export CX_HTTP_PROXY=http://proxy.corp.com:8080
export CX_PROXY_AUTH_TYPE=kerberos
export CX_PROXY_KERBEROS_SPN=HTTP/proxy.corp.com

# 3. Run scan
cx scan create \
  --project-name "MyProject" \
  --source-dir /workspace/myapp \
  --branch main
```

### Mixed Configuration (Environment + Flags)
```bash
# Set proxy in environment
export CX_HTTP_PROXY=http://proxy.company.com:8080

# Use Kerberos with command-line flags
cx scan create \
  --proxy-auth-type kerberos \
  --proxy-kerberos-spn HTTP/proxy.company.com \
  --project-name "MyProject" \
  --source-dir .
```

---

## 🚨 Troubleshooting

### Common NTLM Issues

**Problem**: Authentication fails with 407 Proxy Authentication Required
```
Solution: Verify domain name and credentials
- Check --proxy-ntlm-domain matches your Windows domain
- Ensure username/password in proxy URL are correct
- Test domain format: try both "DOMAIN" and "domain.com"
```

**Problem**: Connection timeout
```
Solution: Check proxy URL format
- Ensure proxy URL includes protocol: http:// or https://
- Verify proxy server address and port are correct
```

### Common Kerberos Issues

**Problem**: "Kerberos SPN is required" error
```
Solution: Always provide the SPN
--proxy-kerberos-spn HTTP/proxy.company.com

Check with your system administrator for the correct SPN format.
```

**Problem**: "Kerberos credential cache not found"
```
Solution: Obtain Kerberos tickets first
kinit username@REALM.COM

Verify tickets exist:
klist
```

**Problem**: "Failed to generate SPNEGO token"
```
Solution: Check SPN format and proxy configuration
- Verify SPN matches proxy server configuration
- Ensure proxy server supports Kerberos authentication
- Check krb5.conf file is properly configured
```

**Problem**: "Kerberos configuration file not found"
```
Solution: Specify krb5.conf location
--proxy-kerberos-krb5-conf /path/to/krb5.conf

Or ensure krb5.conf exists in default location (/etc/krb5.conf)
```

### Testing Authentication

#### Test NTLM
```bash
# Enable verbose logging to see authentication details
cx scan create --verbose \
  --proxy http://user:pass@proxy.com:8080 \
  --proxy-auth-type ntlm \
  --proxy-ntlm-domain DOMAIN \
  --project-name test
```

#### Test Kerberos
```bash
# Enable verbose logging for Kerberos
cx project list create --verbose \
  --proxy http://proxy.com:8080 \
  --proxy-auth-type kerberos \
  --proxy-kerberos-spn HTTP/proxy.com \
  --project-name test
```

---

## 🔒 Security Best Practices

### For NTLM
1. **Use HTTPS proxies** when possible to encrypt credentials
2. **Avoid hardcoding passwords** in scripts - use environment variables
3. **Rotate passwords regularly** according to company policy
4. **Limit proxy access** to necessary users only

### For Kerberos
1. **Secure credential cache** - ensure proper file permissions (600)
2. **Regular ticket renewal** - use kinit periodically for long-running processes
3. **SPN verification** - confirm SPN with proxy administrator
4. **Network security** - ensure Kerberos traffic is protected

### General
1. **Use environment variables** instead of command-line flags for sensitive data
2. **Enable verbose logging** only for troubleshooting
3. **Test authentication** in non-production environments first
4. **Monitor proxy logs** for authentication attempts

---

## 📞 Support

If you encounter issues:

1. **Check logs** with `--verbose` flag
2. **Verify proxy server** supports the chosen authentication method
3. **Contact system administrator** for SPN/domain configuration
4. **Test proxy connectivity** outside of AST CLI first

### System Administrator Checklist

For NTLM:
- [ ] Proxy supports NTLM authentication
- [ ] User account has proxy access permissions
- [ ] Windows domain is correctly configured

For Kerberos:
- [ ] Proxy server has SPN registered in KDC
- [ ] Proxy supports SPNEGO/Kerberos authentication
- [ ] Client machine can reach KDC
- [ ] krb5.conf is properly configured

---

## 📚 Reference

### All Available Flags
```
--proxy                     Proxy server URL
--proxy-auth-type          Authentication type (basic|ntlm|kerberos)
--proxy-ntlm-domain        Windows domain for NTLM
--proxy-kerberos-spn       Service Principal Name for Kerberos
--proxy-kerberos-krb5-conf Path to krb5.conf file
--proxy-kerberos-ccache    Path to Kerberos credential cache
--ignore-proxy             Ignore all proxy settings
```

### All Environment Variables
```
HTTP_PROXY                  Standard proxy environment variable
CX_HTTP_PROXY              Checkmarx proxy URL
CX_PROXY_AUTH_TYPE         Authentication type
CX_PROXY_NTLM_DOMAIN       NTLM domain name
CX_PROXY_KERBEROS_SPN      Kerberos Service Principal Name
CX_PROXY_KERBEROS_KRB5_CONF Kerberos configuration file path
CX_PROXY_KERBEROS_CCACHE   Kerberos credential cache path
KRB5CCNAME                 Standard Kerberos cache environment variable
```


