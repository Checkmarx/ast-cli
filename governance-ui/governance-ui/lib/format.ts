// Display formatters for token and cost figures.
// Unknown is never rendered as zero: callers pass null/undefined and get "—".

export function formatTokens(n: number | null | undefined): string {
  if (n == null) return "—";
  const abs = Math.abs(n);
  if (abs >= 1_000_000_000) return `${(n / 1_000_000_000).toFixed(2)}B`;
  if (abs >= 1_000_000)     return `${(n / 1_000_000).toFixed(2)}M`;
  if (abs >= 10_000)        return `${(n / 1_000).toFixed(1)}K`;
  return n.toLocaleString();
}

export function formatUsd(n: number | null | undefined): string {
  if (n == null) return "—";
  if (n === 0) return "$0.00";
  if (Math.abs(n) < 0.01) return "<$0.01";
  return n.toLocaleString("en-US", { style: "currency", currency: "USD", minimumFractionDigits: 2, maximumFractionDigits: 2 });
}

// A cost that excludes unpriced requests is a lower bound — mark it with "+".
export function formatUsdPartial(cost: number | null | undefined, unpriced: number): string {
  if (cost == null) return "—";
  return unpriced > 0 ? `${formatUsd(cost)}+` : formatUsd(cost);
}
