export type DeviceTagValue =
  | 'router'
  | 'switch'
  | 'access_point'
  | 'firewall'
  | 'printer'
  | 'server'
  | 'workstation'
  | 'nas'
  | 'camera'
  | 'vm_host'
  | 'iot';

export type DeviceTagOption = {
  value: DeviceTagValue;
  label: string;
  description: string;
};

export const DEVICE_TAG_OPTIONS: DeviceTagOption[] = [
  { value: 'router', label: 'Router', description: 'Gateway/router device' },
  { value: 'switch', label: 'Switch', description: 'Layer-2 switch' },
  { value: 'access_point', label: 'Access point', description: 'Wireless AP' },
  { value: 'firewall', label: 'Firewall', description: 'Firewall/VPN edge' },
  { value: 'printer', label: 'Printer', description: 'Printer/MFD' },
  { value: 'server', label: 'Server', description: 'Server or appliance' },
  { value: 'workstation', label: 'Workstation', description: 'PC/laptop' },
  { value: 'nas', label: 'NAS', description: 'Storage/NAS' },
  { value: 'camera', label: 'Camera', description: 'Camera/NVR' },
  { value: 'vm_host', label: 'VM host', description: 'Hypervisor/VM host' },
  { value: 'iot', label: 'IoT', description: 'IoT / embedded device' }
];

export function formatTagLabel(tag: string): string {
  const match = DEVICE_TAG_OPTIONS.find((opt) => opt.value === tag);
  return match?.label ?? tag;
}

