'use client';

import { useEffect } from 'react';
import { useFormState } from 'react-dom';
import { useRouter } from 'next/navigation';

import { createDevice } from './actions';
import { initialCreateDeviceState } from './state';
import { Card, CardBody } from '../../_components/ui/Card';
import { Field, Hint, Label } from '../../_components/ui/Field';
import { Input, Textarea } from '../../_components/ui/Inputs';
import { Button } from '../../_components/ui/Button';
import { Alert } from '../../_components/ui/Alert';

type Props = {
  readOnly?: boolean;
};

export function CreateDeviceForm({ readOnly = false }: Props) {
  const router = useRouter();
  const [state, formAction] = useFormState(createDevice, initialCreateDeviceState());

  useEffect(() => {
    if (state.status === 'success') {
      router.refresh();
    }
  }, [state.status, router]);

  return (
    <form
      action={formAction}
      className="stack"
    >
      <Card style={{ maxWidth: 520 }}>
        <CardBody>
          <div className="stack">
            <div className="split">
              <div>
                <div style={{ fontWeight: 800 }}>Create device</div>
                <div className="hint">Add a device and optional operator metadata.</div>
              </div>
            </div>

            <Field>
              <Label htmlFor="display_name">Display name</Label>
              <Input id="display_name" name="display_name" placeholder="e.g. core switch" disabled={readOnly} />
              <Hint>Optional. You can leave it blank to create an unnamed device.</Hint>
            </Field>

            <Field>
              <Label htmlFor="owner">Owner</Label>
              <Input id="owner" name="owner" placeholder="e.g. network team" disabled={readOnly} />
            </Field>

            <Field>
              <Label htmlFor="location">Location</Label>
              <Input id="location" name="location" placeholder="e.g. data hall A / rack 3" disabled={readOnly} />
            </Field>

            <Field>
              <Label htmlFor="notes">Notes</Label>
              <Textarea id="notes" name="notes" placeholder="free-form notes" rows={3} disabled={readOnly} />
            </Field>

            <div style={{ display: 'flex', gap: 10, alignItems: 'center', flexWrap: 'wrap' }}>
              <Button type="submit" variant="primary" disabled={readOnly}>
                Create device
              </Button>
              {readOnly ? <span className="hint">Read-only access cannot create devices.</span> : null}
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
