import { getData, http } from "./http";

export interface BannedIPItem {
  ip: string;
  ttl_seconds: number;
}

export function getBannedIPs() {
  return getData<{ list: BannedIPItem[] }>(http.get("/security/banned-ips"));
}

export function unbanIP(ip: string) {
  return getData<void>(http.post("/security/banned-ips/unban", { ip }));
}

export default {};
