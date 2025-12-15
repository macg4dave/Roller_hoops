'use client';

import { useRef, useState, type FormEvent } from 'react';
import { useRouter } from 'next/navigation';

type PanelState = {
  status: 'idle' | 'loading' | 'success' | 'error';
  message?: string;
};

const initialState: PanelState = { status: 'idle' };

function messageStyle(state: PanelState) {
  if (state.status === 'error') {
    return { background: '#f9d7da', color: '#b00020' };
  }
  if (state.status === 'success') {
    return { background: '#d1e7dd', color: '#0f5132' };
  }
  return { background: '#eef2ff', color: '#1e3a8a' };
}

function digestImportResponse(payload: string) {
  try {
    const parsed = JSON.parse(payload) as { created?: number; updated?: number; message?: string };
    const parts: string[] = [];
    if (typeof parsed.created === 'number') {
      parts.push(`Created ${parsed.created}`);
    }
    if (typeof parsed.updated === 'number') {
      parts.push(`Updated ${parsed.updated}`);
    }
    if (parsed.message) {
      parts.push(parsed.message);
    }
    return parts.filter(Boolean).join(' · ') || 'Imported devices successfully';
  } catch (error) {
    return 'Imported devices successfully';
  }
}

type Props = {
  readOnly?: boolean;
};

export function ImportExportPanel({ readOnly = false }: Props) {
  const router = useRouter();
  const [state, setState] = useState<PanelState>(initialState);
  const fileInput = useRef<HTMLInputElement>(null);

  const handleExport = async () => {
    setState({ status: 'loading', message: 'Generating snapshot...' });

    try {
      const res = await fetch('/api/devices/export');
      if (!res.ok) {
        const errorBody = await res.text();
        throw new Error(errorBody || `Failed to export (${res.status})`);
      }

      const blob = await res.blob();
      const url = URL.createObjectURL(blob);
      const anchor = document.createElement('a');
      anchor.href = url;
      anchor.download = 'roller_hoops_devices.json';
      document.body.appendChild(anchor);
      anchor.click();
      anchor.remove();
      URL.revokeObjectURL(url);

      setState({ status: 'success', message: 'Snapshot download started.' });
    } catch (error) {
      setState({ status: 'error', message: (error as Error).message || 'Export failed' });
    }
  };

  const handleImport = async (event: FormEvent<HTMLFormElement>) => {
    if (readOnly) {
      setState({ status: 'error', message: 'Read-only users cannot import snapshots.' });
      return;
    }
    event.preventDefault();
    const file = fileInput.current?.files?.[0];
    if (!file) {
      setState({ status: 'error', message: 'Select a JSON snapshot file first.' });
      return;
    }

    setState({ status: 'loading', message: 'Uploading snapshot...' });
    const content = await file.text();

    try {
      const res = await fetch('/api/devices/import', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: content
      });

      const payload = await res.text();
      if (!res.ok) {
        let message = payload;
        try {
          const parsed = JSON.parse(payload);
          message = parsed?.error?.message ?? payload;
        } catch {
          // ignore
        }
        throw new Error(message || `Import failed (${res.status})`);
      }

      setState({ status: 'success', message: digestImportResponse(payload) });
      router.refresh();
    } catch (error) {
      setState({ status: 'error', message: (error as Error).message || 'Import failed' });
    }
  };

  return (
    <section
      style={{
        border: '1px solid #e0e0e0',
        borderRadius: 10,
        padding: 16,
        marginTop: 16,
        display: 'grid',
        gap: 12
      }}
    >
      <div style={{ display: 'flex', justifyContent: 'space-between', gap: 12, flexWrap: 'wrap' }}>
        <div>
          <div style={{ fontSize: 12, letterSpacing: '0.05em', textTransform: 'uppercase', color: '#4b5563' }}>
            Snapshots
          </div>
          <div style={{ color: '#374151', fontSize: 14 }}>
            Export or import the current device catalog as JSON for backups or migrations.
          </div>
        </div>
        <button
          type="button"
          onClick={handleExport}
          style={{
            background: '#111827',
            color: '#fff',
            border: 'none',
            borderRadius: 8,
            padding: '10px 16px',
            fontWeight: 700,
            cursor: 'pointer'
          }}
        >
          Download snapshot
        </button>
      </div>

      <form onSubmit={handleImport} style={{ display: 'grid', gap: 6 }}>
        <label style={{ fontSize: 12, fontWeight: 600, color: '#374151' }}>
          Upload JSON snapshot
        </label>
        <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', alignItems: 'center' }}>
          <input ref={fileInput} type="file" accept="application/json" disabled={readOnly} />
          <button
            type="submit"
            disabled={readOnly}
            style={{
              background: readOnly ? '#9ca3af' : '#111827',
              color: '#fff',
              border: 'none',
              borderRadius: 8,
              padding: '10px 16px',
              fontWeight: 700,
              cursor: readOnly ? 'not-allowed' : 'pointer'
            }}
          >
            Import
          </button>
        </div>
      </form>

      {state.message ? (
        <p
          style={{
            margin: 0,
            padding: '8px 10px',
            borderRadius: 6,
            fontWeight: 600,
            ...messageStyle(state)
          }}
        >
          {state.message}
        </p>
      ) : null}
      {readOnly ? (
        <p style={{ color: '#92400e', fontSize: 13, margin: 0 }}>Read-only access only restricts snapshot imports.</p>
      ) : null}
    </section>
  );
}
