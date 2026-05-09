import { FileTextOutlined, TagsOutlined } from "@ant-design/icons";
import { Typography } from "antd";
import type { ColumnsType } from "antd/es/table";
import { YamlCrudPage } from "../components/k8s/yaml-crud-page";
import { useKeyValueViewer } from "../components/k8s/key-value-viewer";
import { listNamespaces as listClusterNamespaces } from "../services/clusters";
import {
  applyHPA,
  deleteHPA,
  getHPADetail,
  listHPA,
  type HPADetail,
  type HPAItem,
} from "../services/horizontal-pod-autoscalers";

export function HorizontalPodAutoscalersPage() {
  const { renderKVIcon, viewer } = useKeyValueViewer();

  const columns: ColumnsType<HPAItem> = [
    { title: "命名空间", dataIndex: "namespace", width: 120 },
    { title: "名称", dataIndex: "name", width: 200 },
    { title: "伸缩目标", dataIndex: "scale_target_ref", width: 200, ellipsis: true },
    { title: "副本", key: "rep", width: 100, render: (_, r) => `${r.min_replicas ?? "-"}/${r.max_replicas ?? "-"}` },
    {
      title: "指标",
      dataIndex: "metrics_summary",
      width: 220,
      ellipsis: true,
      render: (v?: string) => <Typography.Text style={{ fontSize: 12 }}>{v || "-"}</Typography.Text>,
    },
    {
      title: "条件",
      dataIndex: "conditions_text",
      width: 220,
      ellipsis: true,
      render: (v?: string) => <Typography.Text style={{ fontSize: 12 }}>{v || "-"}</Typography.Text>,
    },
    { title: "标签", key: "labels", width: 70, align: "center", render: (_, r) => renderKVIcon("标签", <TagsOutlined />, r.labels) },
    { title: "存在时长", dataIndex: "age", width: 90 },
    { title: "创建时间", dataIndex: "creation_time", width: 170 },
  ];

  return (
    <>
      <YamlCrudPage<HPAItem, HPADetail>
        title="HPA 弹性伸缩（autoscaling/v2）"
        needNamespace
        onLoadNamespaces={async (cid) => {
          const res = await listClusterNamespaces(cid);
          return (res.list ?? []).map((n) => ({ label: n.name, value: n.name }));
        }}
        columns={columns}
        api={{
          list: async ({ clusterId, namespace, keyword }) => await listHPA(clusterId, namespace ?? "default", keyword),
          detail: async ({ clusterId, namespace, name }) => await getHPADetail(clusterId, namespace ?? "default", name),
          apply: async ({ clusterId, manifest }) => await applyHPA(clusterId, manifest),
          remove: async ({ clusterId, namespace, name }) => await deleteHPA(clusterId, namespace ?? "default", name),
        }}
        createTemplate={({ namespace }) => `apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: demo-hpa
  namespace: ${namespace ?? "default"}
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: demo
  minReplicas: 1
  maxReplicas: 10
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
`}
        detailExtra={(d) => (
          <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
            <FileTextOutlined /> 与 k8m 类似，底层使用 kom/client-go 访问集群；此处为 YAML 管理 HPA。
          </Typography.Paragraph>
        )}
      />
      {viewer}
    </>
  );
}
