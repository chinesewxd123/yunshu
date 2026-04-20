import { getData, http } from "./http";

export interface DictEntryItem {
  id: number;
  dict_type: string;
  label: string;
  value: string;
  sort: number;
  status: number;
  remark?: string;
  created_at: string;
  updated_at: string;
}

export interface DictOptionItem {
  label: string;
  value: string;
}

export interface DictQuery {
  dict_type?: string;
  keyword?: string;
  status?: number;
  page?: number;
  page_size?: number;
}

export interface DictPayload {
  dict_type: string;
  label: string;
  value: string;
  sort?: number;
  status: number;
  remark?: string;
}

export function getDictEntries(params: DictQuery) {
  return getData<{ list: DictEntryItem[]; total: number; page: number; page_size: number }>(
    http.get("/dict/entries", { params }),
  );
}

function normalizeDictPayload(payload: DictPayload): DictPayload {
  const sort = payload.sort == null || Number.isNaN(Number(payload.sort)) ? 0 : Number(payload.sort);
  return { ...payload, sort };
}

export function createDictEntry(payload: DictPayload) {
  return getData<DictEntryItem>(http.post("/dict/entries", normalizeDictPayload(payload)));
}

export function updateDictEntry(id: number, payload: DictPayload) {
  return getData<DictEntryItem>(http.put(`/dict/entries/${id}`, normalizeDictPayload(payload)));
}

export function deleteDictEntry(id: number) {
  return getData<{ message: string }>(http.delete(`/dict/entries/${id}`));
}

export function getDictOptions(dictType: string) {
  return getData<DictOptionItem[]>(http.get(`/dict/options/${dictType}`));
}
