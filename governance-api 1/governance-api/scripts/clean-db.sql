-- Clean all test/dummy data. Keeps schema and the ai_coding_agents catalog.
TRUNCATE policy_actions, session_mcp_servers, ai_sessions,
         developer_agents, mcp_servers, developers, audit_events
RESTART IDENTITY CASCADE;

TRUNCATE policy_versions, policies RESTART IDENTITY CASCADE;

ALTER SEQUENCE pack_version_seq RESTART WITH 1;

SELECT 'Database cleaned — ready for real data.' AS status;
