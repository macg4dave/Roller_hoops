import { redirect } from 'next/navigation';

import { getSessionUser } from '../../lib/auth/session';
import { AppHeader } from '../_components/AppHeader';

export default async function AppLayout({ children }: { children: React.ReactNode }) {
  const session = await getSessionUser();
  if (!session) {
    redirect('/auth/login');
  }

  return (
    <div className="appShell">
      <AppHeader user={session} />
      {session.role === 'read-only' ? (
        <div className="bannerWarning">
          Read-only role: mutation controls are disabled (creating devices, editing metadata, triggering discovery, imports).
        </div>
      ) : null}
      <main className="container">{children}</main>
    </div>
  );
}

