export type DeviceMetadata = {
  owner?: string | null;
  location?: string | null;
  notes?: string | null;
};

export type Device = {
  id: string;
  display_name?: string | null;
  metadata?: DeviceMetadata | null;
};

export type DiscoveryRun = {
  id: string;
  status: string;
  scope?: string | null;
  stats?: Record<string, unknown> | null;
  started_at: string;
  completed_at?: string | null;
  last_error?: string | null;
};

export type DiscoveryStatus = {
  status: string;
  latest_run?: DiscoveryRun | null;
};
