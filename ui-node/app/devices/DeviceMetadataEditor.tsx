'use client';

import { useFormState } from 'react-dom';

import { updateDeviceMetadata } from './actions';
import type { Device } from './types';
import { initialDeviceMetadataState } from './state';

type Props = {
  device: Device;
  readOnly?: boolean;
};

export function DeviceMetadataEditor({ device, readOnly = false }: Props) {
  const [state, formAction] = useFormState(updateDeviceMetadata, initialDeviceMetadataState());

  return (
    <form
      action={formAction}
      style={{
        border: '1px solid #f1f5f9',
        borderRadius: 8,
        padding: 12,
        display: 'grid',
        gap: 8,
        marginTop: 12
      }}
    >
      <input type="hidden" name="device_id" value={device.id} />

      <div style={{ display: 'grid', gap: 6 }}>
        <label style={{ fontSize: 12, fontWeight: 600 }}>Owner</label>
        <input
          name="owner"
          defaultValue={device.metadata?.owner ?? ''}
          placeholder="owner"
          disabled={readOnly}
          style={{ padding: '6px 10px', borderRadius: 6, border: '1px solid #ccc' }}
        />
      </div>

      <div style={{ display: 'grid', gap: 6 }}>
        <label style={{ fontSize: 12, fontWeight: 600 }}>Location</label>
        <input
          name="location"
          defaultValue={device.metadata?.location ?? ''}
          placeholder="location"
          disabled={readOnly}
          style={{ padding: '6px 10px', borderRadius: 6, border: '1px solid #ccc' }}
        />
      </div>

      <div style={{ display: 'grid', gap: 6 }}>
        <label style={{ fontSize: 12, fontWeight: 600 }}>Notes</label>
        <textarea
          name="notes"
          defaultValue={device.metadata?.notes ?? ''}
          placeholder="notes"
          rows={2}
          disabled={readOnly}
          style={{ padding: '6px 10px', borderRadius: 6, border: '1px solid #ccc', resize: 'vertical' }}
        />
      </div>

      <button
        type="submit"
        disabled={readOnly}
        style={{
          background: readOnly ? '#9ca3af' : '#111827',
          color: '#fff',
          border: 'none',
          borderRadius: 6,
          padding: '8px 10px',
          fontWeight: 600,
          cursor: readOnly ? 'not-allowed' : 'pointer',
          alignSelf: 'flex-start'
        }}
      >
        Save metadata
      </button>

      {state.message ? (
        <p
          style={{
            margin: 0,
            color: state.status === 'error' ? '#b00020' : '#0f5132',
            background: state.status === 'error' ? '#f9d7da' : '#d1e7dd',
            borderRadius: 6,
            padding: '6px 10px',
            fontWeight: 600
          }}
        >
          {state.message}
        </p>
      ) : null}
    </form>
  );
}
