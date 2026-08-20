# This file is intentionally vulnerable to trigger devAssist security checks
# It contains ASCA violations that should be detected and blocked

import sqlite3
import subprocess
import os

def authenticate_user(username, password):
    """❌ VULNERABLE: SQL Injection

    This function directly concatenates user input into SQL queries,
    allowing SQL injection attacks.
    """
    db = sqlite3.connect(':memory:')
    cursor = db.cursor()

    # SQL Injection vulnerability: ASCA rule 1001
    query = "SELECT * FROM users WHERE username = '" + username + "' AND password = '" + password + "'"
    cursor.execute(query)

    result = cursor.fetchone()
    return result is not None

def search_logs(search_term, log_file):
    """❌ VULNERABLE: Command Injection

    Unvalidated user input passed directly to shell command.
    """
    # Command injection vulnerability: ASCA rule 2001
    command = f"grep '{search_term}' {log_file} | head -20"
    os.system(command)

def api_configuration():
    """❌ VULNERABLE: Hardcoded Secrets

    Hardcoded API keys and tokens in source code.
    """
    # Hardcoded secret vulnerability: ASCA rule 5002
    API_KEY = "sk-prod-abc123def456ghi789jkl"
    SECRET_TOKEN = "ghp_1234567890abcdefghijklmnopqrstuvwxyz"

    # Another hardcoded credential
    DB_PASSWORD = "admin123!@#SuperSecurePassword"

    return {
        "api_key": API_KEY,
        "token": SECRET_TOKEN,
        "db_password": DB_PASSWORD
    }

def execute_user_command(user_input):
    """❌ VULNERABLE: Unsafe Subprocess

    User-controlled input in subprocess with shell=True.
    """
    # Command injection: ASCA rule 2001
    result = subprocess.call(f"echo {user_input}", shell=True)
    return result

class UserService:
    def __init__(self):
        # More hardcoded secrets
        self.api_token = "token_abc123xyz789"
        self.aws_secret = "AKIA2EXAMPLEAWSSECRET"
        self.db_conn_string = "Server=localhost;User=admin;Password=P@ssw0rd123"

    def fetch_user_data(self, user_id):
        """❌ VULNERABLE: SQL Injection with direct string concatenation"""
        query = f"SELECT * FROM users WHERE user_id = {user_id}"  # ASCA rule 1001
        # Execute without parameterization
        return query

# This file demonstrates multiple ASCA vulnerabilities that devAssist should detect and block
