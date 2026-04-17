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
    <main className="loginPage">
      <div className="loginCard">
        <div className="loginBrand">
          <span className="brandIcon">R</span>
          <h1 className="loginTitle">Sign in to Roller_hoops</h1>
        </div>
        <p className="loginSubtitle">
          Network tracker — sessions are stored in an HTTP-only cookie scoped to this hostname.
        </p>
        <LoginForm />
        <p className="loginHint">
          Configure users via <code>AUTH_USERS</code> in your <code>.env</code> or secret manager.
        </p>
        <p className="loginBack">
          <Link href="/">Back to home</Link>
        </p>
      </div>
    </main>
  );
}
