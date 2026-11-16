export function PageError({ error }: { error: unknown }) {
  return (
    <div className="bg-red-950/40 border border-red-800 rounded-lg p-4 text-sm text-red-400">
      <span className="font-semibold">API error:</span> {String(error)}
    </div>
  );
}

export function EmptyState({ label }: { label: string }) {
  return (
    <div className="border border-gray-800 border-dashed rounded-lg p-10 text-center">
      <p className="text-sm text-gray-500">No {label} recorded yet.</p>
      <p className="text-xs text-gray-600 mt-1">Data will appear here once Claude Code sessions start sending events.</p>
    </div>
  );
}
