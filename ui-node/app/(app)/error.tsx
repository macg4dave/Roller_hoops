"use client";

import { Alert } from '../_components/ui/Alert';
import { Button } from '../_components/ui/Button';

export default function Error({
  error,
  reset
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  return (
    <div className="stack">
      <Alert tone="danger">
        <div style={{ fontWeight: 800 }}>Something went wrong</div>
        <div style={{ marginTop: 6 }}>
          {error.message ? error.message : 'An unexpected error occurred while rendering this page.'}
        </div>
      </Alert>

      <div className="row">
        <Button onClick={() => reset()} variant="primary">
          Try again
        </Button>
      </div>
    </div>
  );
}
