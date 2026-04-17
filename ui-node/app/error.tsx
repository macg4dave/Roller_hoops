"use client";

import Link from 'next/link';

import { Button } from './_components/ui/Button';

export default function Error({
  error
}: {
  error: Error & { digest?: string };
}) {
  return (
    <main className="errorPage">
      <h1 className="errorTitle">Error</h1>
      <p className="errorSubtitle">
        {error.message ? error.message : 'An unexpected error occurred.'}
      </p>
      <Link href="/">
        <Button>Go home</Button>
      </Link>
    </main>
  );
}
