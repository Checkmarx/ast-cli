"use client";
// Token consumption for a single session: totals, composition bar, per-model
// breakdown, sub-agent share and a cumulative-tokens sparkline.
import { useQuery } from "@tanstack/react-query";
import { api, UsageTotals } from "@/lib/api";
import { formatTokens, formatUsd, formatUsdPartial } from "@/lib/format";

// Composition colours — also used as the legend, so they must stay in this order.
const PARTS = [
  { key: "inputTokens",         label: "Input",       color: "#4258c7" },
  { key: "outputTokens",        label: "Output",      color: "#4dbb40" },
  { key: "cacheReadTokens",     label: "Cache read",  color: "#a0ace3" },
  { key: "cacheCreationTokens", label: "Cache write", color: "#f9ae4d" },
] as const;

export function UsageBar({ totals }: { totals: UsageTotals }) {
  const sum = totals.totalTokens || 1;
  return (
    <div>
      <div className="flex h-2 rounded overflow-hidden bg-gray-800" role="img"
           aria-label={PARTS.map((p) => `${p.label} ${formatTokens(totals[p.key])}`).join(", ")}>
        {PARTS.map((p) => (
          <div key={p.key} style={{ width: `${(totals[p.key] / sum) * 100}%`, background: p.color }} />
        ))}
      </div>
      <div className="flex flex-wrap gap-x-4 gap-y-1 mt-2">
        {PARTS.map((p) => (
          <span key={p.key} className="text-xs text-gray-500 flex items-center gap-1.5">
            <span style={{ width: 8, height: 8, borderRadius: 2, background: p.color, display: "inline-block" }} />
            {p.label} <span className="text-gray-300">{formatTokens(totals[p.key])}</span>
          </span>
        ))}
      </div>
    </div>
  );
}

export function Sparkline({ values }: { values: number[] }) {
  if (values.length < 2) return null;
  const W = 300, H = 40, max = Math.max(...values, 1);
  const pts = values.map((v, i) => `${(i / (values.length - 1)) * W},${H - (v / max) * (H - 4) - 2}`).join(" ");
  return (
    <svg viewBox={`0 0 ${W} ${H}`} preserveAspectRatio="none" className="w-full h-10" role="img"
         aria-label="Cumulative tokens over the session">
      <polyline points={pts} fill="none" stroke="#4258c7" strokeWidth="1.5" vectorEffect="non-scaling-stroke" />
    </svg>
  );
}

function Tile({ label, value, sub }: { label: string; value: string; sub?: string }) {
  return (
    <div className="bg-gray-900 border border-gray-800 rounded px-3 py-2">
      <p className="text-[10px] text-gray-500 uppercase tracking-wider">{label}</p>
      <p className="text-lg font-semibold text-gray-200 leading-tight">{value}</p>
      {sub && <p className="text-[10px] text-gray-600">{sub}</p>}
    </div>
  );
}

export function TokenUsageSection({ sessionId }: { sessionId: string }) {
  const { data, isLoading } = useQuery({
    queryKey:        ["session-usage", sessionId],
    queryFn:         () => api.sessionUsage(sessionId),
    refetchInterval: 10_000,
  });

  const heading = <p className="text-xs text-gray-500 uppercase tracking-wider mb-3">Token Usage</p>;

  if (isLoading) return <div className="mb-5">{heading}<p className="text-xs text-gray-600">Loading…</p></div>;
  if (!data?.hasData || !data.totals) {
    return (
      <div className="mb-5">
        {heading}
        <p className="text-xs text-gray-600">
          No token data for this session. Usage is read from the Claude transcript when a turn ends,
          so it needs the governance Stop hook (<span className="font-mono">cx hooks governance install claude</span>).
        </p>
      </div>
    );
  }

  const t = data.totals;
  const byModel = data.byModel ?? [];
  const sub = data.subagents;
  const partial = t.unpricedRequests > 0;

  return (
    <div className="mb-5">
      {heading}
      <div className="grid grid-cols-3 gap-2 mb-3">
        <Tile label="Total tokens" value={formatTokens(t.totalTokens)} sub={`${t.requests} API requests`} />
        <Tile label="Est. cost"    value={formatUsdPartial(t.costUsd, t.unpricedRequests)}
              sub={t.costUsd == null ? "no price for model" : partial ? `${t.unpricedRequests} unpriced` : undefined} />
        <Tile label="Output"       value={formatTokens(t.outputTokens)} />
      </div>
      <UsageBar totals={t} />
      <div className="mt-3"><Sparkline values={(data.series ?? []).map((p) => p.cumulativeTokens)} /></div>

      {byModel.length > 0 && (
        <table className="w-full text-xs mt-3">
          <thead>
            <tr className="text-gray-600 text-left">
              <th className="py-1 font-medium">Model</th>
              <th className="py-1 font-medium text-right">Reqs</th>
              <th className="py-1 font-medium text-right">Tokens</th>
              <th className="py-1 font-medium text-right">Cost</th>
            </tr>
          </thead>
          <tbody>
            {byModel.map((m) => (
              <tr key={m.model} className="border-t border-gray-800">
                <td className="py-1.5 font-mono text-gray-300 truncate max-w-[10rem]" title={m.model}>{m.model}</td>
                <td className="py-1.5 text-right text-gray-400">{m.requests}</td>
                <td className="py-1.5 text-right text-gray-400">{formatTokens(m.totalTokens)}</td>
                <td className="py-1.5 text-right text-gray-300">{formatUsdPartial(m.costUsd, m.unpricedRequests)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
      {sub && sub.requests > 0 && (
        <p className="text-xs text-gray-500 mt-2">
          Sub-agents: {formatTokens(sub.totalTokens)} tokens
          {sub.costUsd != null && <> · {formatUsd(sub.costUsd)}</>} across {sub.requests} requests (included above)
        </p>
      )}
      {partial && (
        <p className="text-xs text-yellow-600 mt-2">
          Cost is a lower bound: {t.unpricedRequests} request(s) use a model with no configured price.
        </p>
      )}
    </div>
  );
}
