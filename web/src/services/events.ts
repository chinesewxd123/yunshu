import { getData, http } from "./http";

export interface EventItem {
  namespace: string;
  type: string;
  reason: string;
  message: string;
  count: number;
  first_time?: string;
  last_time?: string;
  creation_time?: string;
  involved_kind?: string;
  involved_name?: string;
}

export function listEvents(params: {
  cluster_id: number;
  namespace?: string;
  kind?: string;
  name?: string;
  keyword?: string;
  limit?: number;
}) {
  return getData<EventItem[]>(http.get("/events", { params }));
}

