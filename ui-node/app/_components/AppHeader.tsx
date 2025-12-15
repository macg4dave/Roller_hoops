'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';

import { LogoutButton } from './LogoutButton';

type SessionUser = {
  username: string;
  role: string;
};

type NavItem = {
  href: string;
  label: string;
  badge?: string;
};

const NAV_ITEMS: NavItem[] = [
  { href: '/devices', label: 'Devices' },
  { href: '/discovery', label: 'Discovery' },
  { href: '/map', label: 'Map', badge: 'Soon' }
];

function isActivePath(pathname: string, href: string) {
  if (href === '/') return pathname === '/';
  return pathname === href || pathname.startsWith(`${href}/`);
}

export function AppHeader({ user }: { user: SessionUser }) {
  const pathname = usePathname();

  return (
    <header className="appHeader">
      <div className="appHeaderInner">
        <div style={{ display: 'flex', alignItems: 'center', gap: 16, flexWrap: 'wrap' }}>
          <Link href="/devices" className="brand">
            Roller_hoops <span className="brandSubtitle">Network tracker</span>
          </Link>

          <nav className="nav" aria-label="Primary">
            {NAV_ITEMS.map((item) => {
              const active = isActivePath(pathname, item.href);
              const className = active ? 'navLink navLinkActive' : 'navLink';
              return (
                <Link key={item.href} href={item.href} className={className} aria-current={active ? 'page' : undefined}>
                  {item.label}
                  {item.badge ? <span className="navLinkBadge">{item.badge}</span> : null}
                </Link>
              );
            })}
            <Link
              href="/auth/account"
              className={isActivePath(pathname, '/auth/account') ? 'navLink navLinkActive' : 'navLink'}
              aria-current={isActivePath(pathname, '/auth/account') ? 'page' : undefined}
            >
              Account
            </Link>
          </nav>
        </div>

        <div className="headerRight">
          <div className="userMeta">
            <div className="userName">{user.username}</div>
            <div className="userRole">{user.role}</div>
          </div>
          <LogoutButton />
        </div>
      </div>
    </header>
  );
}

