import { useState, useCallback, useEffect, useRef } from "react";
import type { Mapping } from "../api/types";

interface HistoryEntry {
  mapping: Mapping;
}

export function useDesignerState(initial: Mapping | undefined) {
  const [mapping, setMapping] = useState<Mapping>(
    initial || { collections: [] },
  );
  const historyRef = useRef<HistoryEntry[]>([]);
  const redoRef = useRef<HistoryEntry[]>([]);
  const initializedRef = useRef(false);

  // Sync from server/preview when it arrives (only once)
  useEffect(() => {
    if (initial && !initializedRef.current) {
      initializedRef.current = true;
      setMapping(initial);
    }
  }, [initial]);

  const pushHistory = useCallback(() => {
    historyRef.current.push({ mapping: structuredClone(mapping) });
    redoRef.current = [];
  }, [mapping]);

  const undo = useCallback(() => {
    const prev = historyRef.current.pop();
    if (prev) {
      redoRef.current.push({ mapping: structuredClone(mapping) });
      setMapping(prev.mapping);
    }
  }, [mapping]);

  const redo = useCallback(() => {
    const next = redoRef.current.pop();
    if (next) {
      historyRef.current.push({ mapping: structuredClone(mapping) });
      setMapping(next.mapping);
    }
  }, [mapping]);

  const updateMapping = useCallback(
    (updater: (m: Mapping) => Mapping) => {
      pushHistory();
      setMapping((prev) => updater(prev));
    },
    [pushHistory],
  );

  return {
    mapping,
    setMapping,
    updateMapping,
    undo,
    redo,
    canUndo: historyRef.current.length > 0,
    canRedo: redoRef.current.length > 0,
  };
}
