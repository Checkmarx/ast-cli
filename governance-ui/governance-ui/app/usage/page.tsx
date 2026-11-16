"use client";
// Fleet-level token consumption: totals, daily trend, by model, by developer, top sessions.
import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { api, UsageTotals } from "@/lib/api";
import { usePanels } from "@/lib/panel-context";
import { StatCard } from "@/components/stat-card";
import { PageError, EmptyState } from "@/components/page-error";
import { UsageBar } from "@/components/token-usage-section";
import { formatTokens, formatUsd, formatUsdPartial } from "@/lib/format";

const RANGES = [7, 30, 90];

function DailyChart({ days }: { days: { date: string; totalTokens: number; costUsd: number | null }[] }) {
  const max = Math.max(...days.map((d) => d.totalTokens), 1);
  return (
    <div className="flex items-end gap-1 h-32" role="img" aria-label="Tokens per day">
      {days.map((d) => (
        <div key={d.date} className="flex-1 min-w-[6px] flex flex-col justify-end h-full group relative">
          <div className="rounded-t" style={{ height: `${Math.max((d.totalTokens / max) * 100, 2)}%`, background: "#4258c7" }} />
          <div className="hidden group-hover:block absolute bottom-full mb-1 left-1/2 -translate-x-1/2 z-10 whitespace-nowrap
                          bg-gray-800 border border-gray-700 rounded px-2 py-1 text-[11px] text-gray-200">
            {d.date}: {formatTokens(d.totalTokens)} · {formatUsd(d.costUsd)}
          </div>
        </div>
      ))}
    </div>
  );
}

function Cost({ t }: { t: UsageTotals }) {
  return <>{formatUsdPartial(t.costUsd, t.unpricedRequests)}</>;
}

export default function UsagePage() {
  const [days, setDays] = useState(30);
  const { openPanel } = usePanels();
  const { data, isLoading, isError, error } = useQuery({
    queryKey:        ["token-usage", days],
    queryFn:         () => api.tokenUsage(days),
    refetchInterval: 30_000,
  });
  const { data: prices } = useQuery({ queryKey: ["model-prices"], queryFn: api.modelPrices, staleTime: 300_000 });

  const t = data?.totals;
  const empty = !isLoading && !isError && (!t || t.requests === 0);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-lg font-semibold text-gray-200">Token Usage</h1>
        <div className="flex gap-1" role="group" aria-label="Time range">
          {RANGES.map((r) => (
            <button key={r} onClick={() => setDays(r)}
              className={`px-3 py-1 text-xs rounded border ${days === r
                ? "bg-gray-800 border-gray-600 text-gray-200" : "border-gray-800 text-gray-500 hover:text-gray-300"}`}>
              {r}d
            </button>
          ))}
        </div>
      </div>

      {isError && <PageError error={error} />}
      {empty && (
        <>
          <EmptyState label="token usage" />
          <p className="text-xs text-gray-600 text-center">
            Token data comes from the governance Stop hook: <span className="font-mono">cx hooks governance install claude</span>
          </p>
        </>
      )}

      {t && t.requests > 0 && data && (
        <>
          <div className="grid grid-cols-2 xl:grid-cols-4 gap-4">
            <StatCard label="Total Tokens" value={formatTokens(t.totalTokens)} color="blue" sub={`${t.requests.toLocaleString()} API requests`} />
            <StatCard label="Est. Cost" value={formatUsdPartial(t.costUsd, t.unpricedRequests)} color="green"
                      sub={t.unpricedRequests > 0 ? `${t.unpricedRequests} unpriced requests excluded` : "all requests priced"} />
            <StatCard label="Sessions" value={data.sessions} color="yellow" sub={`last ${data.days} days`} />
            <StatCard label="Output Tokens" value={formatTokens(t.outputTokens)} color="blue" sub="incl. thinking" />
          </div>

          <div className="bg-gray-900 border border-gray-800 rounded-lg p-4">
            <p className="text-xs text-gray-500 uppercase tracking-wider mb-3">Composition</p>
            <UsageBar totals={t} />
          </div>

          <div className="bg-gray-900 border border-gray-800 rounded-lg p-4">
            <p className="text-xs text-gray-500 uppercase tracking-wider mb-3">Tokens per day</p>
            <DailyChart days={data.daily} />
          </div>

          <div className="grid grid-cols-1 xl:grid-cols-2 gap-4">
            <div className="bg-gray-900 border border-gray-800 rounded-lg overflow-hidden">
              <p className="px-4 py-3 text-xs text-gray-300 uppercase tracking-wider border-b border-gray-800">By model</p>
              <table className="w-full text-xs">
                <tbody>
                  {data.byModel.map((m) => (
                    <tr key={m.model} className="border-t border-gray-800 first:border-0">
                      <td className="px-4 py-2 font-mono text-gray-300">{m.model}</td>
                      <td className="px-4 py-2 text-right text-gray-400">{formatTokens(m.totalTokens)}</td>
                      <td className="px-4 py-2 text-right text-gray-300"><Cost t={m} /></td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            <div className="bg-gray-900 border border-gray-800 rounded-lg overflow-hidden">
              <p className="px-4 py-3 text-xs text-gray-300 uppercase tracking-wider border-b border-gray-800">By developer</p>
              <table className="w-full text-xs">
                <tbody>
                  {data.byDeveloper.map((d) => (
                    <tr key={d.email} className="border-t border-gray-800 first:border-0">
                      <td className="px-4 py-2 text-gray-300 truncate max-w-[14rem]">{d.email}</td>
                      <td className="px-4 py-2 text-right text-gray-500">{d.sessions} {d.sessions === 1 ? "session" : "sessions"}</td>
                      <td className="px-4 py-2 text-right text-gray-400">{formatTokens(d.totalTokens)}</td>
                      <td className="px-4 py-2 text-right text-gray-300"><Cost t={d} /></td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>

          <div className="bg-gray-900 border border-gray-800 rounded-lg overflow-hidden">
            <p className="px-4 py-3 text-xs text-gray-300 uppercase tracking-wider border-b border-gray-800">Most expensive sessions</p>
            <table className="w-full text-xs">
              <tbody>
                {data.topSessions.map((s) => (
                  <tr key={s.sessionId} onClick={() => openPanel("session", s.sessionId)}
                      className="border-t border-gray-800 first:border-0 hover:bg-gray-800/50 cursor-pointer">
                    <td className="px-4 py-2 text-gray-400">{new Date(s.startedAt).toLocaleString()}</td>
                    <td className="px-4 py-2 text-gray-300">{s.email}</td>
                    <td className="px-4 py-2 text-right text-gray-400">{s.requests} reqs</td>
                    <td className="px-4 py-2 text-right text-gray-400">{formatTokens(s.totalTokens)}</td>
                    <td className="px-4 py-2 text-right text-gray-300"><Cost t={s} /></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}

      {prices && prices.length > 0 && (
        <details className="text-xs text-gray-500">
          <summary className="cursor-pointer hover:text-gray-300">Pricing used ($ per million tokens)</summary>
          <table className="mt-2 w-full border border-gray-800 rounded">
            <thead>
              <tr className="text-gray-600 text-left">
                {["Model prefix", "From", "Input", "Output", "Cache read", "Cache write 5m", "Cache write 1h"].map((h) => (
                  <th key={h} className="px-3 py-1.5 font-medium">{h}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {prices.map((p) => (
                <tr key={p.modelPrefix + p.effectiveFrom} className="border-t border-gray-800">
                  <td className="px-3 py-1.5 font-mono text-gray-300">{p.modelPrefix}</td>
                  <td className="px-3 py-1.5">{p.effectiveFrom}</td>
                  <td className="px-3 py-1.5">{p.inputPerMtok}</td>
                  <td className="px-3 py-1.5">{p.outputPerMtok}</td>
                  <td className="px-3 py-1.5">{p.cacheReadPerMtok}</td>
                  <td className="px-3 py-1.5">{p.cacheWrite5mPerMtok}</td>
                  <td className="px-3 py-1.5">{p.cacheWrite1hPerMtok}</td>
                </tr>
              ))}
            </tbody>
          </table>
          <p className="mt-2">Models without a matching prefix are counted in tokens but not priced.</p>
        </details>
      )}
    </div>
  );
}
