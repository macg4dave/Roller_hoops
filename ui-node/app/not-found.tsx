import Link from 'next/link';

import { Button } from './_components/ui/Button';
import { EmptyState } from './_components/ui/EmptyState';

export default function NotFound() {
  return (
    <main className="container">
      <EmptyState title="Not found">
        <div className="row">
          <div>The page you requested does not exist.</div>
          <Link href="/">
            <Button>Go home</Button>
          </Link>
        </div>
      </EmptyState>
    </main>
  );
}
