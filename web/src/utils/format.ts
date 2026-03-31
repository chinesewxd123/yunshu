import dayjs from "dayjs";

export function formatDateTime(value?: string) {
  if (!value) {
    return "-";
  }
  return dayjs(value).format("YYYY-MM-DD HH:mm:ss");
}

export function statusText(status: number) {
  return status === 1 ? "启用" : "停用";
}

export function statusColor(status: number) {
  return status === 1 ? "success" : "default";
}
