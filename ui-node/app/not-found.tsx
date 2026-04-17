import Link from 'next/link';

import { Button } from './_components/ui/Button';

export default function NotFound() {
  return (
    <main className="errorPage">
      <h1 className="errorTitle">404</h1>
      <p className="errorSubtitle">The page you requested does not exist.</p>
      <Link href="/">
        <Button>Go home</Button>
      </Link>
    </main>
  );
}
