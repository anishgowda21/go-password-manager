export interface KdfPayload {
  type: number;
  time: number;
  memoryKiB: number;
  parallelism: number;
}

export interface SignupRequest {
  email: string;
  authKey: string;
  wrappedVaultKey: string;
  salt: string;
  kdf: KdfPayload;
}
