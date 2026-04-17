import type { components } from '@/lib/api-types';

type Schema<Name extends keyof components['schemas']> = components['schemas'][Name];

export type DeviceMetadata = Schema<'DeviceMetadata'>;
export type Device = Schema<'Device'>;
export type DevicePage = Schema<'DevicePage'>;
export type DeviceFactIP = Schema<'DeviceIP'>;
export type DeviceFactMAC = Schema<'DeviceMAC'>;
export type DeviceFactInterface = Schema<'DeviceInterface'>;
export type DeviceFactService = Schema<'DeviceService'>;
export type DeviceFactSNMP = Schema<'DeviceSNMP'>;
export type DeviceFactLink = Schema<'DeviceLink'>;
export type DeviceFacts = Schema<'DeviceFacts'>;
export type DeviceChangeEvent = Schema<'DeviceChangeEvent'>;
export type DeviceChangeFeed = Schema<'DeviceChangeFeed'>;
export type DeviceNameCandidate = Schema<'DeviceNameCandidate'>;
export type DeviceTag = Schema<'DeviceTag'>;
export type DiscoveryRun = Schema<'DiscoveryRun'>;
export type DiscoveryStatus = Schema<'DiscoveryStatus'>;
export type DiscoveryRunPage = Schema<'DiscoveryRunPage'>;
export type DiscoveryRunLogEntry = Schema<'DiscoveryRunLogEntry'>;
export type DiscoveryRunLogPage = Schema<'DiscoveryRunLogPage'>;
