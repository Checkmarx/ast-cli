"use client";
// Architecture §9.1 — stackable full-screen panels, URL-less, ✕ Close each.

import { usePanels } from "@/lib/panel-context";
import { SessionPanel }   from "./session-panel";
import { ViolationPanel } from "./violation-panel";
import { DeveloperPanel } from "./developer-panel";
import { PolicyPanel }    from "./policy-panel";

export function PanelStack() {
  const { stack, closeTop } = usePanels();
  if (stack.length === 0) return null;

  return (
    <>
      {stack.map((entry, i) => {
        const isTop = i === stack.length - 1;
        return (
          <div
            key={`${entry.type}-${entry.id}-${i}`}
            className="fixed inset-0 overflow-y-auto"
            style={{ background: "var(--background)", zIndex: 50 + i }}
          >
            {entry.type === "session"   && <SessionPanel   id={entry.id} onClose={closeTop} />}
            {entry.type === "violation" && <ViolationPanel id={entry.id} onClose={closeTop} />}
            {entry.type === "developer" && <DeveloperPanel id={entry.id} onClose={closeTop} />}
            {entry.type === "policy"    && <PolicyPanel    id={entry.id} onClose={closeTop} />}
            {/* machine panel — reuses DeveloperPanel scoped to machineId */}
            {entry.type === "machine"   && <DeveloperPanel id={entry.id} onClose={closeTop} machineScope />}
            {/* Suppress unused var warning */ isTop && null}
          </div>
        );
      })}
    </>
  );
}
