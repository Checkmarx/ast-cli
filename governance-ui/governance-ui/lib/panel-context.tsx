"use client";
// Panel context — architecture §9.1: full-screen overlays, stackable, URL-less.

import { createContext, useCallback, useContext, useState } from "react";

export type PanelType = "session" | "developer" | "machine" | "policy" | "violation";

export interface PanelEntry {
  type: PanelType;
  id:   string;
}

interface PanelContextValue {
  stack: PanelEntry[];
  openPanel:  (type: PanelType, id: string) => void;
  closeTop:   () => void;
  closeAll:   () => void;
}

const PanelCtx = createContext<PanelContextValue>({
  stack: [],
  openPanel: () => {},
  closeTop:  () => {},
  closeAll:  () => {},
});

export function PanelProvider({ children }: { children: React.ReactNode }) {
  const [stack, setStack] = useState<PanelEntry[]>([]);

  const openPanel = useCallback((type: PanelType, id: string) => {
    setStack((s) => [...s, { type, id }]);
  }, []);

  const closeTop = useCallback(() => {
    setStack((s) => s.slice(0, -1));
  }, []);

  const closeAll = useCallback(() => setStack([]), []);

  return (
    <PanelCtx.Provider value={{ stack, openPanel, closeTop, closeAll }}>
      {children}
    </PanelCtx.Provider>
  );
}

export function usePanels() {
  return useContext(PanelCtx);
}
