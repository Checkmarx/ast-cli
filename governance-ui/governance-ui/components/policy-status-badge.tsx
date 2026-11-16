"use client";

interface Props {
  status:     string; // "active" | "disabled"
  policyType: string; // "governance" | "pii" | "third_party_monitor"
}

export function PolicyStatusBadge({ status, policyType }: Props) {
  if (status === "disabled") {
    return (
      <span className="px-2 py-0.5 rounded text-xs font-medium bg-gray-800 text-gray-500 border border-gray-700">
        Disabled
      </span>
    );
  }
  if (policyType === "third_party_monitor") {
    return (
      <span className="px-2 py-0.5 rounded text-xs font-medium bg-orange-900/30 text-orange-400 border border-orange-800">
        Third-Party Monitor
      </span>
    );
  }
  if (policyType === "pii") {
    return (
      <span className="px-2 py-0.5 rounded text-xs font-medium bg-purple-900/30 text-purple-400 border border-purple-800">
        PII
      </span>
    );
  }
  return (
    <span className="px-2 py-0.5 rounded text-xs font-medium bg-green-900/30 text-green-400 border border-green-800">
      Active
    </span>
  );
}
