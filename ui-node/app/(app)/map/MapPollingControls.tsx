'use client';

import { Button } from '@/app/_components/ui/Button';

import { useMapProjection } from './MapProjectionContext';
import { useMapLayout } from './MapLayoutContext';

export function MapPollingControls() {
  const { pinned, setPinned, hasPendingUpdates, applyPendingUpdates, refreshing } = useMapProjection();
  const { requestAutoLayout } = useMapLayout();

  return (
    <div className="mapCanvasControls" aria-label="Map projection controls">
      <Button type="button" onClick={requestAutoLayout}>
        Auto layout
      </Button>
      {hasPendingUpdates ? (
        <Button type="button" variant="primary" onClick={applyPendingUpdates}>
          Apply updates
        </Button>
      ) : null}
      <Button type="button" onClick={() => setPinned(!pinned)}>
        {pinned ? 'Enable live' : 'Pause live'}
      </Button>
      <span className="mapCanvasControlsHint" aria-live="polite">
        {refreshing ? 'Refreshing…' : pinned ? 'Live updates paused.' : 'Live updates enabled.'}
      </span>
    </div>
  );
}
