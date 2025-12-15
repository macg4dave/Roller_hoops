import Link from 'next/link';
import { redirect } from 'next/navigation';

import { getSessionUser } from '../../../../lib/auth/session';
import { AccountSettings } from './AccountSettings';

export default async function AccountPage() {
  const session = await getSessionUser();
  if (!session) {
    redirect('/auth/login');
  }
  return (
    <main style={{ display: 'grid', gap: 16 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', gap: 12, flexWrap: 'wrap', alignItems: 'baseline' }}>
        <h1 style={{ fontSize: 28, margin: 0 }}>Account</h1>
        <Link href="/devices" style={{ color: '#111827' }}>
          Back to devices
        </Link>
      </div>
      <AccountSettings username={session.username} role={session.role} />
    </main>
  );
}
