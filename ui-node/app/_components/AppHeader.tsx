'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';

import { LogoutButton } from './LogoutButton';
import { api } from '@/lib/api-client';
import { Badge } from './ui/Badge';
import { getDiscoveryStatusBadgeTone } from '../(app)/discovery/status';

import type { DiscoveryStatus } from '../(app)/devices/types';

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

function DiscoveryHeaderIndicator() {
  const statusQuery = useQuery({
    queryKey: ['discovery-status'],
    queryFn: async ({ signal }) => {
      const res = await api.GET('/v1/discovery/status', {
        signal,
        headers: {
          'X-Request-ID': globalThis.crypto?.randomUUID?.()
        }
      });

      if (res.error) {
        throw new Error('Failed to fetch discovery status.');
      }
      return (res.data ?? { status: 'unknown', latest_run: null }) as DiscoveryStatus;
    },
    refetchInterval: () => {
      if (typeof document !== 'undefined' && document.visibilityState === 'hidden') {
        return false;
      }
      return 10_000;
    },
    refetchIntervalInBackground: false
  });

  const status = statusQuery.data?.status;
  const inProgress = status === 'queued' || status === 'running';

  if (!inProgress) return null;

  return (
    <Link
      href="/discovery"
      className="headerIndicator"
      aria-label={`Discovery ${status} — view details`}
      title={`Discovery ${status}`}
    >
      <span className="headerIndicatorLabel">Discovery</span>
      <Badge tone={getDiscoveryStatusBadgeTone(status)}>{status}</Badge>
    </Link>
  );
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
          <DiscoveryHeaderIndicator />
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

