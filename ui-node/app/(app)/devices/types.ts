export type DeviceMetadata = {
  owner?: string | null;
  location?: string | null;
  notes?: string | null;
};

export type Device = {
  id: string;
  display_name?: string | null;
  primary_ip?: string | null;
  last_seen_at?: string | null;
  last_change_at?: string | null;
  metadata?: DeviceMetadata | null;
};

export type DevicePage = {
  devices: Device[];
  cursor?: string | null;
};

export type DeviceFactIP = {
  ip: string;
  interface_id?: string | null;
  interface_name?: string | null;
  created_at: string;
  updated_at: string;
};

export type DeviceFactMAC = {
  mac: string;
  interface_id?: string | null;
  interface_name?: string | null;
  created_at: string;
  updated_at: string;
};

export type DeviceFactInterface = {
  id: string;
  name?: string | null;
  ifindex?: number | null;
  descr?: string | null;
  alias?: string | null;
  mac?: string | null;
  admin_status?: number | null;
  oper_status?: number | null;
  mtu?: number | null;
  speed_bps?: number | null;
  pvid?: number | null;
  pvid_observed_at?: string | null;
  created_at: string;
  updated_at: string;
};

export type DeviceFactService = {
  protocol?: 'tcp' | 'udp' | null;
  port?: number | null;
  name?: string | null;
  state?: 'open' | 'closed' | null;
  source?: string | null;
  observed_at: string;
  created_at: string;
  updated_at: string;
};

export type DeviceFactSNMP = {
  address?: string | null;
  sys_name?: string | null;
  sys_descr?: string | null;
  sys_object_id?: string | null;
  sys_contact?: string | null;
  sys_location?: string | null;
  last_success_at?: string | null;
  last_error?: string | null;
  updated_at: string;
};

export type DeviceFactLink = {
  id: string;
  link_key: string;
  peer_device_id: string;
  local_interface_id?: string | null;
  peer_interface_id?: string | null;
  link_type?: string | null;
  source: string;
  observed_at?: string | null;
  updated_at: string;
};

export type DeviceFacts = {
  device_id: string;
  ips: DeviceFactIP[];
  macs: DeviceFactMAC[];
  interfaces: DeviceFactInterface[];
  services: DeviceFactService[];
  snmp?: DeviceFactSNMP | null;
  links: DeviceFactLink[];
};

export type DeviceChangeEvent = {
  event_id: string;
  device_id: string;
  event_at: string;
  kind: string;
  summary: string;
  details?: Record<string, unknown> | null;
};

export type DeviceChangeFeed = {
  events: DeviceChangeEvent[];
  cursor?: string | null;
};

export type DeviceNameCandidate = {
  name: string;
  source: string;
  address?: string | null;
  observed_at: string;
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
