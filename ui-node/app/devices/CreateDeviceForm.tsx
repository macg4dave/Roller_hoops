'use client';

import { useEffect } from 'react';
import { useFormState } from 'react-dom';
import { useRouter } from 'next/navigation';

import { createDevice } from './actions';
import { initialCreateDeviceState } from './state';

export function CreateDeviceForm() {
  const router = useRouter();
  const [state, formAction] = useFormState(createDevice, initialCreateDeviceState());

  useEffect(() => {
    if (state.status === 'success') {
      router.refresh();
    }
  }, [state.status, router]);

  return (
    <form
      action={formAction}
      style={{
        border: '1px solid #e0e0e0',
        borderRadius: 8,
        padding: 16,
        marginTop: 16,
        maxWidth: 480,
        display: 'grid',
        gap: 12
      }}
    >
      <div>
        <label style={{ display: 'block', fontWeight: 600 }} htmlFor="display_name">
          Display name
        </label>
        <input
          id="display_name"
          name="display_name"
          placeholder="e.g. core switch"
          style={{
            marginTop: 6,
            padding: '10px 12px',
            width: '100%',
            borderRadius: 6,
            border: '1px solid #ccc'
          }}
        />
        <p style={{ color: '#666', marginTop: 6, fontSize: 14 }}>
          Optional. You can leave it blank to create an unnamed device.
        </p>
      </div>

      <button
        type="submit"
        style={{
          background: '#0f766e',
          color: '#fff',
          border: 'none',
          borderRadius: 6,
          padding: '10px 12px',
          fontWeight: 600,
          cursor: 'pointer',
          width: 'fit-content'
        }}
      >
        Create device
      </button>

      {state.message ? (
        <p
          style={{
            margin: 0,
            color: state.status === 'error' ? '#b00020' : '#0f5132',
            background: state.status === 'error' ? '#f9d7da' : '#d1e7dd',
            borderRadius: 6,
            padding: '8px 10px',
            fontWeight: 600
          }}
        >
          {state.message}
        </p>
      ) : null}
    </form>
  );
}
