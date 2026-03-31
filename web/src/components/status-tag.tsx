import { Tag } from "antd";
import { statusColor, statusText } from "../utils/format";

interface StatusTagProps {
  status: number;
}

export function StatusTag({ status }: StatusTagProps) {
  return <Tag color={statusColor(status)}>{statusText(status)}</Tag>;
}
