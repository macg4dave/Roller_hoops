'use client';

import { createContext, useCallback, useContext, useMemo, useState } from 'react';

type MapLayoutContextValue = {
  autoLayoutToken: number;
  requestAutoLayout: () => void;
};

const MapLayoutContext = createContext<MapLayoutContextValue | null>(null);

export function MapLayoutProvider({ children }: { children: React.ReactNode }) {
  const [autoLayoutToken, setAutoLayoutToken] = useState(0);

  const requestAutoLayout = useCallback(() => {
    setAutoLayoutToken((prev) => prev + 1);
  }, []);

  const value = useMemo<MapLayoutContextValue>(
    () => ({
      autoLayoutToken,
      requestAutoLayout
    }),
    [autoLayoutToken, requestAutoLayout]
  );

  return <MapLayoutContext.Provider value={value}>{children}</MapLayoutContext.Provider>;
}

export function useMapLayout(): MapLayoutContextValue {
  const context = useContext(MapLayoutContext);
  if (!context) {
    throw new Error('useMapLayout must be used within a MapLayoutProvider');
  }
  return context;
}

