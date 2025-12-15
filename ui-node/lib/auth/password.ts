import { randomBytes, scryptSync, timingSafeEqual } from 'crypto';

const DEFAULT_SCRYPT = {
  N: 16384,
  r: 8,
  p: 1,
  keyLen: 32
} as const;

type ScryptHash = {
  N: number;
  r: number;
  p: number;
  saltHex: string;
  hashHex: string;
};

export function hashPasswordScrypt(password: string) {
  const salt = randomBytes(16);
  const hash = scryptSync(password, salt, DEFAULT_SCRYPT.keyLen, {
    cost: DEFAULT_SCRYPT.N,
    blockSize: DEFAULT_SCRYPT.r,
    parallelization: DEFAULT_SCRYPT.p
  });

  const saltHex = salt.toString('hex');
  const hashHex = Buffer.from(hash).toString('hex');
  return `scrypt$${DEFAULT_SCRYPT.N}$${DEFAULT_SCRYPT.r}$${DEFAULT_SCRYPT.p}$${saltHex}$${hashHex}`;
}

export function verifyPasswordScrypt(password: string, encoded: string) {
  const parsed = parseScrypt(encoded);
  if (!parsed) {
    return false;
  }
  const salt = Buffer.from(parsed.saltHex, 'hex');
  const expected = Buffer.from(parsed.hashHex, 'hex');
  const actual = scryptSync(password, salt, expected.length, {
    cost: parsed.N,
    blockSize: parsed.r,
    parallelization: parsed.p
  });
  return expected.length === actual.length && timingSafeEqual(expected, Buffer.from(actual));
}

function parseScrypt(encoded: string): ScryptHash | null {
  const parts = encoded.split('$');
  if (parts.length !== 6) {
    return null;
  }
  if (parts[0] !== 'scrypt') {
    return null;
  }
  const N = Number(parts[1]);
  const r = Number(parts[2]);
  const p = Number(parts[3]);
  const saltHex = parts[4];
  const hashHex = parts[5];
  if (!Number.isFinite(N) || !Number.isFinite(r) || !Number.isFinite(p) || N <= 1 || r <= 0 || p <= 0) {
    return null;
  }
  if (!saltHex || !hashHex) {
    return null;
  }
  return { N, r, p, saltHex, hashHex };
}

