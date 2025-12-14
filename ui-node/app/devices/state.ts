export type CreateDeviceState = {
  status: 'idle' | 'success' | 'error';
  message?: string;
};

const defaultState: CreateDeviceState = { status: 'idle' };

export function initialCreateDeviceState(): CreateDeviceState {
  return { ...defaultState };
}

export type DiscoveryRunState = {
  status: 'idle' | 'success' | 'error';
  message?: string;
};

export function initialDiscoveryRunState(): DiscoveryRunState {
  return { status: 'idle' };
}
