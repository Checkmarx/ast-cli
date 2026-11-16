-- Update session hook for 16ac3e3b-1d46-4477-a8c5-ee5fe672167b
-- Replace PowerShell command block with aegis third-party hook

UPDATE session_hooks
SET
  command = 'aegis',
  vendor = 'aegis',
  event = 'third_party_hook'
WHERE session_id = '16ac3e3b-1d46-4477-a8c5-ee5fe672167b'
  AND command LIKE '%powershell%'
  OR command LIKE '%PowerShell%';

-- Verify the update
SELECT
  id,
  session_id,
  event,
  matcher,
  command,
  vendor,
  created_at
FROM session_hooks
WHERE session_id = '16ac3e3b-1d46-4477-a8c5-ee5fe672167b';
