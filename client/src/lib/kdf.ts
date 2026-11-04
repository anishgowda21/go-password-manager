import type { KdfPayload } from "./types";

export const KDF_TYPE_ARGON2ID = 1;

export const KDF_DEFAULTS = {
  type: KDF_TYPE_ARGON2ID,
  time: 3,
  memoryKiB: 64 * 1024,
  parallelism: 1,
} as const;

export const KDF_BOUNDS = {
  time: [1, 5],
  memoryKiB: [16 * 1024, 64 * 1024],
  parallelism: [1, 4],
} as const;

function checkRange(
  name: string,
  value: number,
  [min, max]: readonly [number, number],
) {
  if (!Number.isInteger(value) || value < min || value > max) {
    throw new Error(`Invalid KDF ${name}: ${value} (allowed ${min}-${max})`);
  }
}

export function validateKdf(k: KdfPayload): void {
  if (k.type !== KDF_TYPE_ARGON2ID) {
    throw new Error(`Unsupported KDF type: ${k.type}`);
  }
  checkRange("time", k.time, KDF_BOUNDS.time);
  checkRange("memoryKiB", k.memoryKiB, KDF_BOUNDS.memoryKiB);
  checkRange("parallelism", k.parallelism, KDF_BOUNDS.parallelism);
}
