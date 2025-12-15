"use client";

import Link from 'next/link';

import { Alert } from './_components/ui/Alert';
import { Button } from './_components/ui/Button';

export default function Error({
  error
}: {
  error: Error & { digest?: string };
}) {
  return (
    <main className="container">
      <div className="stack">
        <Alert tone="danger">
          <div style={{ fontWeight: 800 }}>Something went wrong</div>
          <div style={{ marginTop: 6 }}>
            {error.message ? error.message : 'An unexpected error occurred.'}
          </div>
        </Alert>

        <Link href="/">
          <Button>Go home</Button>
        </Link>
      </div>
    </main>
  );
}
