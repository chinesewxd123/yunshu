import { Button, Drawer, Modal, Table, Tooltip, Typography } from "antd";
import { useState } from "react";

type ViewerMode = "modal" | "drawer";

type UseKeyValueViewerOptions = {
  mode?: ViewerMode;
  title?: string;
  emptyText?: string | ((title: string) => string);
  width?: number;
  compact?: boolean;
  pageSize?: number;
  destroyOnClose?: boolean;
};

export function useKeyValueViewer(options?: UseKeyValueViewerOptions) {
  const mode = options?.mode ?? "modal";
  const emptyText = options?.emptyText ?? "暂无数据";
  const width = options?.width ?? 720;
  const compact = options?.compact ?? false;
  const pageSize = options?.pageSize;
  const destroyOnClose = options?.destroyOnClose ?? false;
  const defaultTitle = options?.title ?? "详情";

  const [open, setOpen] = useState(false);
  const [title, setTitle] = useState(defaultTitle);
  const [data, setData] = useState<Record<string, string>>({});

  const openKV = (nextTitle: string, nextData?: Record<string, string>) => {
    setTitle(nextTitle);
    setData(nextData ?? {});
    setOpen(true);
  };

  const renderKVIcon = (nextTitle: string, icon: JSX.Element, nextData?: Record<string, string>) => (
    <Tooltip title={nextTitle}>
      <Button type="link" size="small" icon={icon} onClick={() => openKV(nextTitle, nextData)} />
    </Tooltip>
  );

  const content = (
    <Table
      rowKey={(r) => r.key}
      size={compact ? "small" : "middle"}
      pagination={typeof pageSize === "number" && pageSize > 0 ? { pageSize } : false}
      dataSource={Object.entries(data).map(([key, value]) => ({ key, value }))}
      locale={{ emptyText: typeof emptyText === "function" ? emptyText(title) : emptyText }}
      columns={[
        { title: "Key", dataIndex: "key", width: 260, render: (v: string) => <Typography.Text copyable>{v}</Typography.Text> },
        { title: "Value", dataIndex: "value", render: (v: string) => <Typography.Text copyable style={{ whiteSpace: "pre-wrap" }}>{v}</Typography.Text> },
      ]}
    />
  );

  const viewer =
    mode === "drawer" ? (
      <Drawer title={title} open={open} onClose={() => setOpen(false)} width={width}>
        {content}
      </Drawer>
    ) : (
      <Modal title={title} open={open} onCancel={() => setOpen(false)} footer={null} width={width} destroyOnClose={destroyOnClose}>
        {content}
      </Modal>
    );

  return { openKV, renderKVIcon, viewer };
}
