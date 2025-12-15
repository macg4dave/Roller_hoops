'use client';

import { useFormState } from 'react-dom';

import { updateDeviceMetadata } from './actions';
import type { Device } from './types';
import { initialDeviceMetadataState } from './state';
import { Card, CardBody } from '../../_components/ui/Card';
import { Field, Label } from '../../_components/ui/Field';
import { Input, Textarea } from '../../_components/ui/Inputs';
import { Button } from '../../_components/ui/Button';
import { Alert } from '../../_components/ui/Alert';

type Props = {
  device: Device;
  readOnly?: boolean;
};

export function DeviceMetadataEditor({ device, readOnly = false }: Props) {
  const [state, formAction] = useFormState(updateDeviceMetadata, initialDeviceMetadataState());

  return (
    <form action={formAction} className="stack">
      <input type="hidden" name="device_id" value={device.id} />

      <Card>
        <CardBody>
          <div className="stack">
            <div style={{ fontWeight: 800 }}>Metadata</div>

            <Field>
              <Label>Owner</Label>
              <Input name="owner" defaultValue={device.metadata?.owner ?? ''} placeholder="owner" disabled={readOnly} />
            </Field>

            <Field>
              <Label>Location</Label>
              <Input
                name="location"
                defaultValue={device.metadata?.location ?? ''}
                placeholder="location"
                disabled={readOnly}
              />
            </Field>

            <Field>
              <Label>Notes</Label>
              <Textarea
                name="notes"
                defaultValue={device.metadata?.notes ?? ''}
                placeholder="notes"
                rows={2}
                disabled={readOnly}
              />
            </Field>

            <div style={{ display: 'flex', gap: 10, alignItems: 'center', flexWrap: 'wrap' }}>
              <Button type="submit" variant="primary" disabled={readOnly}>
                Save metadata
              </Button>
              {readOnly ? <span className="hint">Read-only access cannot edit metadata.</span> : null}
            </div>

            {state.message ? (
              <Alert tone={state.status === 'error' ? 'danger' : 'success'}>{state.message}</Alert>
            ) : null}
          </div>
        </CardBody>
      </Card>
    </form>
  );
}
