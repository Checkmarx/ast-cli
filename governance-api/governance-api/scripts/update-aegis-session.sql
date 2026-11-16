-- Delete the old session with unknown IDE
DELETE FROM ai_sessions WHERE session_id = '1cc41448-05e7-4deb-93eb-45933ad40417';

-- Update the aegis session to use npm install lodash instead of PowerShell
DO $$ DECLARE
  v_session_id UUID := 'b3b2db58-c7a3-4e09-a4f2-c0b686047e31'::UUID;
BEGIN
  -- Update activity events to use npm instead of PowerShell
  UPDATE activity_events
  SET tool_name = 'npm',
      tool_command = 'install lodash',
      tool_category = 'package_manager'
  WHERE session_id = v_session_id AND tool_name = 'PowerShell';

  -- Update policy actions to reflect npm package manager instead of shell
  UPDATE policy_actions
  SET resource_type = 'package_manager',
      resource_id = 'npm install lodash',
      reason = 'Aegis - Third Party Hook Blocked: npm install lodash',
      metadata = '{"vendor": "aegis", "reason": "third_party_enforcement", "description": "npm install lodash blocked by Aegis", "tool": "npm"}'::jsonb,
      violation_type = 'command'
  WHERE session_id = v_session_id;

  RAISE NOTICE 'Updated aegis session % to use npm install lodash', v_session_id;
END $$;

-- Show final session state
SELECT
  s.session_id,
  s.claude_session_id,
  s.ide_name,
  s.status,
  d.email,
  a.agent_name,
  s.started_at,
  (SELECT COUNT(*) FROM policy_actions WHERE session_id = s.session_id) as block_count,
  (SELECT COUNT(*) FROM activity_events WHERE session_id = s.session_id) as activity_count
FROM ai_sessions s
JOIN developers d ON s.developer_id = d.developer_id
JOIN ai_coding_agents a ON s.agent_id = a.agent_id
WHERE d.email = 'anjali.deore@checkmarx.com'
ORDER BY s.started_at DESC;

-- Show activity details for aegis session
SELECT
  kind,
  action_type,
  tool_name,
  tool_command,
  policy_name,
  occurred_at
FROM activity_events
WHERE session_id = 'b3b2db58-c7a3-4e09-a4f2-c0b686047e31'
ORDER BY occurred_at;
