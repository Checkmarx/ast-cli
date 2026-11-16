"use client";
import { useQuery } from "@tanstack/react-query";
import { api, AuditEvent } from "@/lib/api";
import { PageError, EmptyState } from "@/components/page-error";

export default function AuditPage() {
  const { data, isLoading, isError, error } = useQuery({
    queryKey: ["audit"],
    queryFn:  () => api.auditLog(),
  });
  const events: AuditEvent[] = data ?? [];

  return (
    <div>
      <h1 className="text-lg font-semibold text-gray-200 mb-2">Audit Log</h1>
      <p className="text-xs text-gray-600 mb-6">
        Append-only log of system mutations — policy changes, developer changes.
      </p>

      {isError && <PageError error={error} />}

      {!isLoading && !isError && events.length === 0 && (
        <EmptyState label="audit events" />
      )}

      {events.length > 0 && (
        <div className="border border-gray-800 rounded-lg overflow-hidden">
          <table className="w-full text-xs">
            <thead>
              <tr className="bg-gray-900 text-gray-500 uppercase tracking-wider">
                {["Timestamp","Event Type","Resource","Actor"].map((h) => (
                  <th key={h} className="px-4 py-3 text-left font-medium">{h}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {events.map((e) => (
                <tr key={e.auditEventId} className="border-t border-gray-800 hover:bg-gray-900">
                  <td className="px-4 py-3 text-gray-500">{new Date(e.timestamp).toLocaleString()}</td>
                  <td className="px-4 py-3 text-gray-300 font-semibold">{e.eventType}</td>
                  <td className="px-4 py-3 text-gray-500">{e.resourceType || "—"}</td>
                  <td className="px-4 py-3 text-gray-500">{e.actorEmail || e.actorType}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
