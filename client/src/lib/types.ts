export interface KdfPayload {
  type: number;
  time: number;
  memoryKiB: number;
  parallelism: number;
}

export interface SignupRequest {
  email: string;
  deviceId: string;
  deviceName: string;
  authKey: string;
  wrappedVaultKey: string;
  salt: string;
  kdf: KdfPayload;
}

export interface PreloginResponse {
  salt: string;
  kdf: KdfPayload;
}

export interface LoginRequest {
  email: string;
  authKey: string;
  deviceId: string;
  deviceName: string;
}

export interface AuthResponse {
  userId: string;
  accessToken: string;
  refreshToken: string;
  expiresIn: number;
  wrappedVaultKey: string;
  salt: string;
  kdf: KdfPayload;
}
