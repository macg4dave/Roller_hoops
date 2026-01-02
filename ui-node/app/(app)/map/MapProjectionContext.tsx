'use client';

import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';

import type { components } from '@/lib/api-types';
import { api } from '@/lib/api-client';

type MapLayer = components['schemas']['MapLayer'];
type MapFocusType = components['schemas']['MapFocusType'];
type MapProjection = components['schemas']['MapProjection'];

type MapFocus = {
  type: MapFocusType;
  id: string;
};

type MapProjectionContextValue = {
  layer?: MapLayer;
  focus?: MapFocus;
  pinned: boolean;
  setPinned: (pinned: boolean) => void;
  projection?: MapProjection;
  pendingProjection?: MapProjection;
  hasPendingUpdates: boolean;
  applyPendingUpdates: () => void;
  refreshing: boolean;
};

const MapProjectionContext = createContext<MapProjectionContextValue | null>(null);

function stableStringify(value: unknown): string {
  if (value === null) return 'null';
  if (value === undefined) return 'undefined';
  if (typeof value === 'string') return JSON.stringify(value);
  if (typeof value === 'number' || typeof value === 'boolean') return String(value);
  if (Array.isArray(value)) {
    return `[${value.map((entry) => stableStringify(entry)).join(',')}]`;
  }
  if (typeof value === 'object') {
    const record = value as Record<string, unknown>;
    const keys = Object.keys(record).sort((a, b) => a.localeCompare(b));
    return `{${keys.map((key) => `${JSON.stringify(key)}:${stableStringify(record[key])}`).join(',')}}`;
  }
  return JSON.stringify(value);
}

function projectionFingerprint(projection: MapProjection): string {
  return stableStringify({
    layer: projection.layer,
    focus: projection.focus,
    guidance: projection.guidance,
    regions: projection.regions,
    nodes: projection.nodes,
    edges: projection.edges,
    inspector: projection.inspector,
    truncation: projection.truncation
  });
}

async function fetchProjection({
  layer,
  focus,
  signal
}: {
  layer: MapLayer;
  focus: MapFocus;
  signal?: AbortSignal;
}): Promise<MapProjection> {
  const res = await api.GET('/v1/map/{layer}', {
    params: {
      path: { layer },
      query: { focusType: focus.type, focusId: focus.id }
    },
    signal,
    headers: {
      'X-Request-ID': globalThis.crypto?.randomUUID?.()
    }
  });

  if (res.error || !res.data) {
    throw new Error('Failed to fetch map projection.');
  }

  return res.data as MapProjection;
}

export function MapProjectionProvider({
  children,
  layer,
  focus,
  initialProjection
}: {
  children: React.ReactNode;
  layer?: MapLayer;
  focus?: MapFocus;
  initialProjection?: MapProjection;
}) {
  const [pinned, setPinned] = useState(true);
  const [appliedProjection, setAppliedProjection] = useState<MapProjection | undefined>(initialProjection);
  const [pendingProjection, setPendingProjection] = useState<MapProjection | undefined>(undefined);

  const enabled = Boolean(layer && focus);

  const projectionQuery = useQuery({
    queryKey: ['map-projection', layer, focus?.type, focus?.id],
    enabled,
    initialData: initialProjection,
    queryFn: ({ signal }) => fetchProjection({ layer: layer as MapLayer, focus: focus as MapFocus, signal }),
    refetchInterval: (query) => {
      if (typeof document !== 'undefined' && document.visibilityState === 'hidden') {
        return false;
      }
      return query.state.status === 'error' ? 30_000 : 10_000;
    },
    refetchIntervalInBackground: false,
    refetchOnMount: false
  });

  const latestProjection = projectionQuery.data ?? initialProjection;

  const appliedFingerprint = useMemo(() => {
    if (!appliedProjection) return null;
    return projectionFingerprint(appliedProjection);
  }, [appliedProjection]);

  const latestFingerprint = useMemo(() => {
    if (!latestProjection) return null;
    return projectionFingerprint(latestProjection);
  }, [latestProjection]);

  useEffect(() => {
    if (!latestProjection) {
      return;
    }
    if (!appliedProjection) {
      setAppliedProjection(latestProjection);
      return;
    }
    if (latestFingerprint && appliedFingerprint && latestFingerprint === appliedFingerprint) {
      setPendingProjection(undefined);
      return;
    }

    if (pinned) {
      setPendingProjection(latestProjection);
      return;
    }

    setAppliedProjection(latestProjection);
    setPendingProjection(undefined);
  }, [latestProjection, pinned, latestFingerprint, appliedFingerprint, appliedProjection]);

  const hasPendingUpdates = Boolean(pendingProjection && latestFingerprint && appliedFingerprint && latestFingerprint !== appliedFingerprint);

  const applyPendingUpdates = useCallback(() => {
    if (!pendingProjection) {
      return;
    }
    setAppliedProjection(pendingProjection);
    setPendingProjection(undefined);
  }, [pendingProjection]);

  const setPinnedWithSemantics = useCallback(
    (nextPinned: boolean) => {
      setPinned(nextPinned);
      if (!nextPinned && pendingProjection) {
        setAppliedProjection(pendingProjection);
        setPendingProjection(undefined);
      }
    },
    [pendingProjection]
  );

  const value = useMemo<MapProjectionContextValue>(
    () => ({
      layer,
      focus,
      pinned,
      setPinned: setPinnedWithSemantics,
      projection: appliedProjection,
      pendingProjection,
      hasPendingUpdates,
      applyPendingUpdates,
      refreshing: projectionQuery.isFetching
    }),
    [
      layer,
      focus,
      pinned,
      setPinnedWithSemantics,
      appliedProjection,
      pendingProjection,
      hasPendingUpdates,
      applyPendingUpdates,
      projectionQuery.isFetching
    ]
  );

  return <MapProjectionContext.Provider value={value}>{children}</MapProjectionContext.Provider>;
}

export function useMapProjection(): MapProjectionContextValue {
  const context = useContext(MapProjectionContext);
  if (!context) {
    throw new Error('useMapProjection must be used within a MapProjectionProvider');
  }
  return context;
}

export function useOptionalMapProjection(): MapProjectionContextValue | null {
  return useContext(MapProjectionContext);
}
