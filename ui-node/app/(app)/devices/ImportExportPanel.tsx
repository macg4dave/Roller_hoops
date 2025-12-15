'use client';

import { useRef, useState, type FormEvent } from 'react';
import { useRouter } from 'next/navigation';

import { Alert } from '@/app/_components/ui/Alert';
import { Button } from '@/app/_components/ui/Button';
import { Card, CardBody } from '@/app/_components/ui/Card';
import { Field, Label, Hint } from '@/app/_components/ui/Field';

type PanelState = {
  status: 'idle' | 'loading' | 'success' | 'error';
  message?: string;
};

const initialState: PanelState = { status: 'idle' };

function messageTone(state: PanelState) {
  if (state.status === 'error') return 'danger' as const;
  if (state.status === 'success') return 'success' as const;
  if (state.status === 'loading') return 'info' as const;
  return 'info' as const;
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
    <Card>
      <CardBody className="stack">
        <div className="split">
          <div className="stack" style={{ gap: 6 }}>
            <p className="kicker">Snapshots</p>
            <div className="hint">Export or import the current device catalog as JSON for backups or migrations.</div>
          </div>
          <Button type="button" onClick={handleExport} disabled={state.status === 'loading'}>
            Download snapshot
          </Button>
        </div>

        <form onSubmit={handleImport} className="stack" style={{ gap: 8 }}>
          <Field>
            <Label>Upload JSON snapshot</Label>
            <div className="row">
              <input ref={fileInput} type="file" accept="application/json" disabled={readOnly} className="fileInput" />
              <Button type="submit" variant="primary" disabled={readOnly || state.status === 'loading'}>
                Import
              </Button>
            </div>
            <Hint>Import will upsert devices by ID.</Hint>
          </Field>
        </form>

        {state.message ? <Alert tone={messageTone(state)}>{state.message}</Alert> : null}
        {readOnly ? <Alert tone="warning">Read-only access prevents snapshot imports.</Alert> : null}
      </CardBody>
    </Card>
  );
}
