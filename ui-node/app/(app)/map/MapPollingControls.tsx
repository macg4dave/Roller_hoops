'use client';

import { Badge } from '@/app/_components/ui/Badge';
import { Button } from '@/app/_components/ui/Button';

import { useMapProjection } from './MapProjectionContext';

export function MapPollingControls() {
  const { pinned, setPinned, hasPendingUpdates, applyPendingUpdates, refreshing } = useMapProjection();

  return (
    <div className="mapCanvasControls" aria-label="Map projection controls">
      <Badge tone={pinned ? 'neutral' : 'success'}>{pinned ? 'Pinned' : 'Live'}</Badge>
      {hasPendingUpdates ? <Badge tone="warning">Updates available</Badge> : null}
      {hasPendingUpdates ? (
        <Button type="button" variant="primary" onClick={applyPendingUpdates}>
          Apply updates
        </Button>
      ) : null}
      <Button type="button" onClick={() => setPinned(!pinned)}>
        {pinned ? 'Unpin' : 'Pin focus'}
      </Button>
      <span className="mapCanvasControlsHint" aria-live="polite">
        {refreshing
          ? 'Refreshing…'
          : pinned
            ? hasPendingUpdates
              ? 'Updates queued until applied.'
              : 'Updates will queue while pinned.'
            : 'Updates apply automatically.'}
      </span>
    </div>
  );
}

