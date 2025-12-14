export type CreateDeviceState = {
  status: 'idle' | 'success' | 'error';
  message?: string;
};

const defaultState: CreateDeviceState = { status: 'idle' };

export function initialCreateDeviceState(): CreateDeviceState {
  return { ...defaultState };
}
