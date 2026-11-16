-- Create Anjali developer entry
INSERT INTO developers (email, display_name, username, status)
VALUES ('anjali.deore@checkmarx.com', 'Anjali Deore', 'anjali.deore', 'active')
ON CONFLICT (email) DO UPDATE SET status = 'active';

-- Create the session with aegis block
DO $$ DECLARE
  v_dev_id UUID;
  v_agent_id UUID;
  v_session_id UUID;
  v_policy_id UUID;
BEGIN
  -- Get developer
  SELECT developer_id INTO v_dev_id FROM developers WHERE email = 'anjali.deore@checkmarx.com' LIMIT 1;

  -- Get agent
  SELECT agent_id INTO v_agent_id FROM ai_coding_agents WHERE agent_name = 'Claude Code' LIMIT 1;

  -- Create session with IDE info and ACTIVE status
  v_session_id := gen_random_uuid();
  INSERT INTO ai_sessions (session_id, claude_session_id, developer_id, agent_id, started_at, status, ide_name, ide_version, workspace_path)
  VALUES (
    v_session_id,
    'claude-' || substr(md5(gen_random_bytes(16)::text), 1, 16),
    v_dev_id,
    v_agent_id,
    NOW(),
    'active',
    'vscode',
    '1.93.0',
    'C:\Users\AnjaliD\Documents\MyProjects\AI Governance'
  );

  -- Session start event
  INSERT INTO activity_events (activity_event_id, event_id, session_id, kind, agent_type, occurred_at)
  VALUES (gen_random_uuid(), 'event-' || gen_random_uuid()::text, v_session_id, 'session.start', 'agent', NOW());

  -- Prompt submission event - create PowerShell script request
  INSERT INTO activity_events (activity_event_id, event_id, session_id, kind, agent_type, prompt_length, action_type, occurred_at)
  VALUES (gen_random_uuid(), 'event-' || gen_random_uuid()::text, v_session_id, 'prompt.submit', 'agent', 65, 'ALLOW', NOW());

  -- Tool execution attempt - PowerShell (tool.pre)
  INSERT INTO activity_events (activity_event_id, event_id, session_id, kind, agent_type, tool_category, tool_name, tool_command, action_type, occurred_at)
  VALUES (gen_random_uuid(), 'event-' || gen_random_uuid()::text, v_session_id, 'tool.pre', 'agent', 'shell', 'PowerShell', 'Export-AWSCredentials', 'ALLOW', NOW());

  -- Get a policy
  SELECT policy_id INTO v_policy_id FROM policies LIMIT 1;
  IF v_policy_id IS NULL THEN
    INSERT INTO policies (policy_name, description) VALUES ('Shell Block Policy', 'Block shell execution') RETURNING policy_id INTO v_policy_id;
  END IF;

  -- Tool post - AEGIS BLOCKED the PowerShell execution
  INSERT INTO activity_events (
    activity_event_id, event_id, session_id, kind, agent_type, tool_category,
    tool_name, tool_command, action_type, policy_name, policy_id,
    matched_value, occurred_at, raw
  )
  VALUES (
    gen_random_uuid(),
    'event-' || gen_random_uuid()::text,
    v_session_id,
    'tool.post',
    'agent',
    'shell',
    'PowerShell',
    'Export-AWSCredentials',
    'BLOCK',
    'Aegis Third-Party Hook',
    v_policy_id::text,
    'powershell.exe',
    NOW(),
    '{"vendor": "aegis", "enforcement": "third_party_hook", "action": "BLOCK", "reason": "Aegis detected third-party hook attempting PowerShell execution"}'::jsonb
  );

  -- Policy action: Aegis blocked - PowerShell execution blocked
  INSERT INTO policy_actions (
    policy_action_id, session_id, policy_id, developer_id, agent_id,
    action_type, violation_type, severity, resource_type, resource_id,
    reason, occurred_at, resolution_status, metadata
  )
  VALUES (
    gen_random_uuid(),
    v_session_id,
    v_policy_id,
    v_dev_id,
    v_agent_id,
    'BLOCK',
    'command',
    'high',
    'shell',
    'powershell.exe',
    'Aegis - Third Party Hook Blocked',
    NOW(),
    'open',
    '{"vendor": "aegis", "reason": "third_party_enforcement", "description": "PowerShell execution blocked by Aegis"}'::jsonb
  );

  RAISE NOTICE 'Successfully created session % with VSCODE IDE and AEGIS BLOCK', v_session_id;
END $$;

-- Show all sessions with their recent activity
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
ORDER BY s.started_at DESC
LIMIT 5;
