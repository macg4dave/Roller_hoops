'use client';

import { useEffect, useMemo, useRef, useState } from 'react';
import { useFormState } from 'react-dom';
import { useRouter } from 'next/navigation';

import { Alert } from '@/app/_components/ui/Alert';
import { Badge } from '@/app/_components/ui/Badge';
import { Button } from '@/app/_components/ui/Button';
import { Card, CardBody } from '@/app/_components/ui/Card';
import { EmptyState } from '@/app/_components/ui/EmptyState';

import type { DeviceTag } from './types';
import { updateDeviceTags } from './actions';
import { initialDeviceTagsState } from './state';
import { DEVICE_TAG_OPTIONS, formatTagLabel, type DeviceTagValue } from './tags';

type Props = {
  deviceId: string;
  readOnly?: boolean;
};

function isTagValue(value: string): value is DeviceTagValue {
  return DEVICE_TAG_OPTIONS.some((opt) => opt.value === value);
}

function normalizeTag(value: string) {
  return value.trim().toLowerCase();
}

export function DeviceTagsPanel({ deviceId, readOnly = false }: Props) {
  const router = useRouter();
  const [state, formAction] = useFormState(updateDeviceTags, initialDeviceTagsState());
  const [tags, setTags] = useState<DeviceTag[] | null>(null);
  const [loading, setLoading] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [refreshKey, setRefreshKey] = useState(0);
  const [manualTags, setManualTags] = useState<DeviceTagValue[]>([]);
  const hasInitializedRef = useRef(false);

  useEffect(() => {
    let cancelled = false;
    async function load() {
      setLoading(true);
      setLoadError(null);
      try {
        const res = await fetch(`/api/devices/${deviceId}/tags`, { cache: 'no-store' });
        if (res.status === 404) {
          throw new Error('Device not found.');
        }
        if (!res.ok) {
          throw new Error(`Request failed (${res.status})`);
        }
        const body = (await res.json()) as DeviceTag[];
        if (!cancelled) {
          setTags(body);
        }
      } catch (err) {
        if (!cancelled) {
          setTags(null);
          setLoadError(err instanceof Error ? err.message : 'Failed to load tags');
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }
    void load();
    return () => {
      cancelled = true;
    };
  }, [deviceId, refreshKey]);

  useEffect(() => {
    if (state.status === 'success') {
      router.refresh();
      setRefreshKey((v) => v + 1);
    }
  }, [router, state.status]);

  useEffect(() => {
    if (!tags) {
      return;
    }
    if (hasInitializedRef.current) {
      return;
    }
    hasInitializedRef.current = true;
    const selected: DeviceTagValue[] = [];
    for (const tag of tags) {
      if (tag.source !== 'manual') continue;
      const normalized = normalizeTag(tag.tag);
      if (isTagValue(normalized)) {
        selected.push(normalized);
      }
    }
    setManualTags(selected);
  }, [tags]);

  const effectiveTags = useMemo(() => {
    if (!tags) return [];
    const out: string[] = [];
    const seen = new Set<string>();
    for (const tag of tags) {
      const normalized = normalizeTag(tag.tag);
      if (!normalized) continue;
      if (seen.has(normalized)) continue;
      seen.add(normalized);
      out.push(normalized);
    }
    return out;
  }, [tags]);

  const manualTagSet = useMemo(() => new Set(manualTags), [manualTags]);
  const autoTags = useMemo(() => (tags ?? []).filter((t) => t.source === 'auto'), [tags]);

  const statusTone = state.status === 'error' ? 'danger' : state.status === 'success' ? 'success' : 'info';

  return (
    <Card>
      <CardBody className="stack" style={{ gap: 10 }}>
        <div className="split" style={{ alignItems: 'baseline' }}>
          <div className="stack" style={{ gap: 6 }}>
            <p className="kicker">Tags</p>
            <div className="hint">Auto-tags are best-effort and can be overridden manually.</div>
          </div>
          <Button type="button" onClick={() => setRefreshKey((v) => v + 1)} disabled={loading}>
            Refresh
          </Button>
        </div>

        {loading ? <div className="hint">Loading…</div> : null}
        {loadError ? <Alert tone="danger">{loadError}</Alert> : null}

        {effectiveTags.length === 0 && !loading ? (
          <EmptyState title="No tags yet">
            Run discovery with enrichment (SNMP/ports/names) to generate suggestions, or set manual tags below.
          </EmptyState>
        ) : effectiveTags.length > 0 ? (
          <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
            {effectiveTags.slice(0, 10).map((tag) => (
              <Badge key={tag} tone={manualTagSet.has(tag as DeviceTagValue) ? 'success' : 'neutral'}>
                {formatTagLabel(tag)}
              </Badge>
            ))}
            {effectiveTags.length > 10 ? <span className="hint">+{effectiveTags.length - 10} more</span> : null}
          </div>
        ) : null}

        <form action={formAction} className="stack" style={{ gap: 10 }}>
          <input type="hidden" name="device_id" value={deviceId} />
          {manualTags.map((tag) => (
            <input key={tag} type="hidden" name="tags" value={tag} />
          ))}

          <div className="stack" style={{ gap: 8 }}>
            <div style={{ fontWeight: 800 }}>Manual override</div>
            <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
              {DEVICE_TAG_OPTIONS.map((opt) => {
                const active = manualTags.includes(opt.value);
                return (
                  <Button
                    key={opt.value}
                    type="button"
                    className={`btnPill${active ? ' btnPillActive' : ''}`}
                    aria-pressed={active}
                    disabled={readOnly}
                    onClick={() => {
                      setManualTags((prev) => {
                        if (prev.includes(opt.value)) {
                          return prev.filter((t) => t !== opt.value);
                        }
                        return [...prev, opt.value];
                      });
                    }}
                    title={opt.description}
                  >
                    {opt.label}
                  </Button>
                );
              })}
            </div>
            <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
              <Button type="submit" variant="primary" disabled={readOnly}>
                Save tags
              </Button>
              <Button type="button" disabled={readOnly || manualTags.length === 0} onClick={() => setManualTags([])}>
                Clear manual tags
              </Button>
              {readOnly ? <span className="hint">Read-only access cannot edit tags.</span> : null}
            </div>
          </div>
        </form>

        {autoTags.length > 0 ? (
          <details className="hint">
            <summary style={{ cursor: 'pointer', fontWeight: 750 }}>Auto tag evidence</summary>
            <div style={{ marginTop: 8, display: 'grid', gap: 8 }}>
              {autoTags.slice(0, 12).map((tag) => (
                <div key={`${tag.tag}-${tag.source}-${tag.updated_at}`} className="card" style={{ padding: 10, borderRadius: 10 }}>
                  <div className="split" style={{ alignItems: 'baseline' }}>
                    <strong>{formatTagLabel(tag.tag)}</strong>
                    <span className="hint">Confidence {tag.confidence}</span>
                  </div>
                  {tag.evidence ? (
                    <pre className="hint" style={{ margin: '6px 0 0', whiteSpace: 'pre-wrap' }}>
                      {JSON.stringify(tag.evidence, null, 2)}
                    </pre>
                  ) : (
                    <div className="hint" style={{ marginTop: 6 }}>
                      No evidence recorded.
                    </div>
                  )}
                </div>
              ))}
            </div>
          </details>
        ) : null}

        {state.message ? <Alert tone={statusTone}>{state.message}</Alert> : null}
      </CardBody>
    </Card>
  );
}

