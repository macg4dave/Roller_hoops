import '@testing-library/jest-dom';

import { render, screen } from '@testing-library/react';
import { describe, expect, test, vi } from 'vitest';

import { ConfirmDialog } from './ConfirmDialog';

describe('ConfirmDialog', () => {
  test('renders as a modal dialog and focuses the cancel button', async () => {
    const onCancel = vi.fn();
    const onConfirm = vi.fn();

    render(
      <ConfirmDialog
        open
        title="Confirm thing"
        description="Are you sure?"
        confirmLabel="Do it"
        cancelLabel="Nope"
        onCancel={onCancel}
        onConfirm={onConfirm}
      />
    );

    expect(screen.getByRole('dialog', { name: 'Confirm thing' })).toBeInTheDocument();

    const cancel = screen.getByRole('button', { name: 'Nope' });
    // focus is set in an effect
    await new Promise((r) => setTimeout(r, 0));
    expect(document.activeElement).toBe(cancel);
  });
});
