const DEVICE_ID_KEY = "deviceId";

function browserName(): string {
  const ua = navigator.userAgent;
  if (ua.includes("Edg/")) return "Edge";
  if (ua.includes("OPR/")) return "Opera";
  if (ua.includes("Firefox/")) return "Firefox";
  if (ua.includes("Chrome/")) return "Chrome";
  if (ua.includes("Safari/")) return "Safari";
  return "browser";
}

export function getDeviceInfo() {
  let deviceId = localStorage.getItem(DEVICE_ID_KEY);
  if (!deviceId) {
    deviceId = crypto.randomUUID();
    localStorage.setItem(DEVICE_ID_KEY, deviceId);
  }
  return { deviceId, deviceName: browserName() };
}
