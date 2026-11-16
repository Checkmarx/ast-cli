-- Run this as the postgres superuser ONCE after installing PostgreSQL:
-- psql -U postgres -f scripts/setup-db.sql

-- Create the governance database and user
CREATE USER governance WITH PASSWORD 'governance';
CREATE DATABASE governance OWNER governance;
GRANT ALL PRIVILEGES ON DATABASE governance TO governance;

-- Connect to the governance database
\c governance

-- Grant schema privileges
GRANT ALL ON SCHEMA public TO governance;
