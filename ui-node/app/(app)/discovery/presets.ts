export type ScanPreset = 'fast' | 'normal' | 'deep';

export type ScanPresetOption = {
  value: ScanPreset;
  label: string;
  description: string;
};

export const SCAN_PRESET_OPTIONS: ScanPresetOption[] = [
  {
    value: 'fast',
    label: 'Fast',
    description: 'Quick sweep with tighter budgets; disables SNMP and service scanning.'
  },
  {
    value: 'normal',
    label: 'Normal',
    description: 'Balanced defaults (recommended).'
  },
  {
    value: 'deep',
    label: 'Deep',
    description: 'Thorough sweep with higher budgets; enables SNMP and service scanning when configured.'
  }
];

export function getScanPresetLabel(value: unknown): string {
  if (typeof value !== 'string') return 'Normal';
  const normalized = value.trim().toLowerCase();
  const match = SCAN_PRESET_OPTIONS.find((p) => p.value === normalized);
  return match?.label ?? 'Normal';
}

export function normalizeScanPreset(value: FormDataEntryValue | null | undefined): ScanPreset {
  if (typeof value !== 'string') return 'normal';
  const normalized = value.trim().toLowerCase();
  if (normalized === 'fast' || normalized === 'deep') return normalized;
  return 'normal';
}

