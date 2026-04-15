import { Drawer, Table, Tag } from "antd";
import { useState } from "react";

type PodsTarget = {
  clusterId: number;
  namespace: string;
  name: string;
};

type RelatedPodItem = {
  namespace?: string;
  name?: string;
  phase?: string;
  node_name?: string;
  pod_ip?: string;
  restart_count?: number;
  start_time?: string;
};

type UseRelatedPodsDrawerOptions = {
  width?: number;
  titlePrefix?: string;
};

export function useRelatedPodsDrawer(
  fetchPods: (target: PodsTarget) => Promise<RelatedPodItem[] | undefined>,
  options?: UseRelatedPodsDrawerOptions,
) {
  const width = options?.width ?? 900;
  const titlePrefix = options?.titlePrefix ?? "关联 Pods";

  const [open, setOpen] = useState(false);
  const [loading, setLoading] = useState(false);
  const [target, setTarget] = useState<PodsTarget | null>(null);
  const [pods, setPods] = useState<RelatedPodItem[]>([]);

  const openPods = (nextTarget: PodsTarget) => {
    setTarget(nextTarget);
    setOpen(true);
    setLoading(true);
    void (async () => {
      try {
        const items = await fetchPods(nextTarget);
        setPods(items ?? []);
      } finally {
        setLoading(false);
      }
    })();
  };

  const viewer = (
    <Drawer title={`${titlePrefix}${target ? `：${target.name}` : ""}`} open={open} onClose={() => setOpen(false)} width={width}>
      <Table
        rowKey={(r) => `${r.namespace}/${r.name}`}
        loading={loading}
        dataSource={pods}
        pagination={{ pageSize: 10 }}
        columns={[
          { title: "Pod 名称", dataIndex: "name" },
          { title: "状态", dataIndex: "phase", width: 120, render: (v: string) => <Tag color={v === "Running" ? "green" : "default"}>{v || "-"}</Tag> },
          { title: "节点", dataIndex: "node_name", width: 160 },
          { title: "PodIP", dataIndex: "pod_ip", width: 140 },
          { title: "重启", dataIndex: "restart_count", width: 90 },
          { title: "启动时间", dataIndex: "start_time", width: 180 },
        ]}
      />
    </Drawer>
  );

  return { openPods, viewer };
}
