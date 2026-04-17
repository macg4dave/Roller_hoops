import { redirect } from 'next/navigation';

import { getSessionUser } from '../../../../lib/auth/session';
import { AccountSettings } from './AccountSettings';

export default async function AccountPage() {
  const session = await getSessionUser();
  if (!session) {
    redirect('/auth/login');
  }
  return (
    <section className="stack">
      <header>
        <h1 className="pageTitle">Account</h1>
        <p className="pageSubTitle">
          Manage your session and view your current role.
        </p>
      </header>
      <AccountSettings username={session.username} role={session.role} />
    </section>
  );
}
