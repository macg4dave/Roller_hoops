import Link from 'next/link';
import { redirect } from 'next/navigation';
import { getSessionUser } from '../../../lib/auth/session';
import { LoginForm } from './LoginForm';

export default async function LoginPage() {
  const currentUser = await getSessionUser();
  if (currentUser) {
    redirect('/devices');
  }

  return (
    <main
      style={{
        display: 'grid',
        justifyItems: 'center',
        paddingTop: 48
      }}
    >
      <div
        style={{
          width: 'min(420px, 90vw)',
          border: '1px solid #e5e7eb',
          borderRadius: 12,
          padding: 24,
          textAlign: 'left'
        }}
      >
        <h1 style={{ margin: 0, fontSize: 24 }}>Sign in to Roller_hoops</h1>
        <p style={{ color: '#4b5563', margin: '6px 0 16px' }}>
          The UI owns authentication. Sessions are stored in an HTTP-only cookie scoped to this hostname.
        </p>
        <LoginForm />
        <p style={{ marginTop: 18, color: '#6b7280', fontSize: 13 }}>
          Configure users via `AUTH_USERS` (or back-compat `AUTH_USERNAME` / `AUTH_PASSWORD`) in your `.env` or secret manager.
        </p>
        <p style={{ marginTop: 4, fontSize: 13 }}>
          <Link href="/" style={{ color: '#2563eb' }}>
            Back to home
          </Link>
        </p>
      </div>
    </main>
  );
}
