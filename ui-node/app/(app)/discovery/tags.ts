export type ScanTag = 'ports' | 'snmp' | 'topology' | 'names';

export type ScanTagOption = {
  value: ScanTag;
  label: string;
  description: string;
};

export const SCAN_TAG_OPTIONS: ScanTagOption[] = [
  {
    value: 'ports',
    label: 'Ports (nmap)',
    description: 'Run TCP port scanning via nmap when installed and allowlisted.'
  },
  {
    value: 'snmp',
    label: 'SNMP',
    description: 'Poll SNMP for sysName/sysDescr and interface/VLAN enrichment.'
  },
  {
    value: 'topology',
    label: 'Topology',
    description: 'Attempt LLDP/CDP neighbor collection (requires SNMP + topology allowlist).'
  },
  {
    value: 'names',
    label: 'Names',
    description: 'Try reverse DNS/mDNS name hints for better labels.'
  }
];

const TAG_VALUES = new Set<ScanTag>(SCAN_TAG_OPTIONS.map((opt) => opt.value));

export function normalizeScanTags(values: FormDataEntryValue[] | null | undefined): ScanTag[] {
  if (!values || values.length === 0) {
    return [];
  }

  const out: ScanTag[] = [];
  for (const value of values) {
    if (typeof value !== 'string') {
      continue;
    }
    const normalized = value.trim().toLowerCase();
    if (!normalized) {
      continue;
    }
    if (!TAG_VALUES.has(normalized as ScanTag)) {
      continue;
    }
    if (!out.includes(normalized as ScanTag)) {
      out.push(normalized as ScanTag);
    }
  }
  return out;
}

export function formatScanTags(value: unknown): string | null {
  if (!value) {
    return null;
  }
  if (!Array.isArray(value)) {
    return null;
  }
  const tags = normalizeScanTags(value as FormDataEntryValue[]);
  if (!tags.length) {
    return null;
  }
  const labelByValue = new Map(SCAN_TAG_OPTIONS.map((opt) => [opt.value, opt.label] as const));
  return tags.map((tag) => labelByValue.get(tag) ?? tag).join(', ');
}

