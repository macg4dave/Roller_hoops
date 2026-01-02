'use client';

import { createContext, useContext, useMemo, useState } from 'react';

export type MapSelection =
  | { kind: 'region'; id: string }
  | { kind: 'node'; id: string }
  | null;

type MapSelectionContextValue = {
  selection: MapSelection;
  setSelection: (selection: MapSelection) => void;
  clearSelection: () => void;
};

const MapSelectionContext = createContext<MapSelectionContextValue | null>(null);

export function MapSelectionProvider({ children }: { children: React.ReactNode }) {
  const [selection, setSelection] = useState<MapSelection>(null);

  const value = useMemo<MapSelectionContextValue>(
    () => ({
      selection,
      setSelection,
      clearSelection: () => setSelection(null)
    }),
    [selection]
  );

  return <MapSelectionContext.Provider value={value}>{children}</MapSelectionContext.Provider>;
}

export function useMapSelection(): MapSelectionContextValue {
  const context = useContext(MapSelectionContext);
  if (!context) {
    throw new Error('useMapSelection must be used within a MapSelectionProvider');
  }
  return context;
}

