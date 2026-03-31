import type { MessageData, PolicyItem, PolicyPayload } from "../types/api";
import { getData, http } from "./http";

export function getPolicies() {
  return getData<PolicyItem[]>(http.get("/policies"));
}

export function grantPolicy(payload: PolicyPayload) {
  return getData<MessageData>(http.post("/policies", payload));
}

export function revokePolicy(payload: PolicyPayload) {
  return getData<MessageData>(http.delete("/policies", { data: payload }));
}