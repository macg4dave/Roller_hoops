export type BadgeTone = 'neutral' | 'success' | 'warning' | 'danger' | 'info';

export function getDiscoveryStatusBadgeTone(status: string): BadgeTone {
  switch (status) {
    case 'running':
      return 'warning';
    case 'queued':
      return 'info';
    case 'succeeded':
      return 'success';
    case 'failed':
      return 'danger';
    default:
      return 'neutral';
  }
}