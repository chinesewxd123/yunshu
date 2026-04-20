import { FileTextOutlined, TagsOutlined } from "@ant-design/icons";
import { Button, Modal, Typography, message } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useRef } from "react";
import { IngressFormCreateDrawer } from "../components/k8s/k8s-resource-form-drawers";
import { useKeyValueViewer } from "../components/k8s/key-value-viewer";
import { YamlCrudPage } from "../components/k8s/yaml-crud-page";
import { listNamespaces as listClusterNamespaces } from "../services/clusters";
import { applyIngress, deleteIngress, getIngressDetail, listIngresses, restartIngressNginxPods, type IngressDetail, type IngressItem } from "../services/ingresses";

export function IngressesPage() {
  const listReloadRef = useRef<() => void>(() => {});
  const { renderKVIcon, viewer } = useKeyValueViewer();

  const columns: ColumnsType<IngressItem> = [
    { title: "命名空间", dataIndex: "namespace", width: 120 },
    { title: "名称", dataIndex: "name", width: 180 },
    {
      title: "访问规则",
      dataIndex: "rules_text",
      width: 320,
      render: (v?: string) => <Typography.Text style={{ whiteSpace: "pre-wrap", fontSize: 12 }}>{v || "-"}</Typography.Text>,
    },
    { title: "标签", key: "labels", width: 70, align: "center", render: (_, r) => renderKVIcon("标签", <TagsOutlined />, r.labels) },
    { title: "注解", key: "annotations", width: 70, align: "center", render: (_, r) => renderKVIcon("注解", <FileTextOutlined />, r.annotations) },
    { title: "入口控制器", dataIndex: "class_name", width: 180, render: (v?: string) => v || "-" },
    { title: "LB地址", dataIndex: "load_balancer", width: 180, render: (v?: string) => v || "-" },
    { title: "存在时长", dataIndex: "age", width: 90, fixed: "right", render: (v?: string) => v || "-" },
    { title: "创建时间", dataIndex: "creation_time", width: 180, fixed: "right" },
  ];

  return (
    <>
      <YamlCrudPage<IngressItem, IngressDetail>
        title="Ingress-Nginx 管理"
        needNamespace
        onLoadNamespaces={async (cid) => {
          const res = await listClusterNamespaces(cid);
          return (res.list ?? []).map((n) => ({ label: n.name, value: n.name }));
        }}
        columns={columns}
        api={{
          list: async ({ clusterId, namespace, keyword }) => await listIngresses(clusterId, namespace ?? "default", keyword),
          detail: async ({ clusterId, namespace, name }) => await getIngressDetail(clusterId, namespace ?? "default", name),
          apply: async ({ clusterId, manifest }) => await applyIngress(clusterId, manifest),
          remove: async ({ clusterId, namespace, name }) => await deleteIngress(clusterId, namespace ?? "default", name),
        }}
        onToolbarReady={(ctx) => {
          listReloadRef.current = ctx.reload;
        }}
        renderCreateFormTab={(ctx) => (
          <IngressFormCreateDrawer
            embedded
            open
            clusterId={ctx.clusterId}
            namespace={ctx.namespace ?? "default"}
            onClose={ctx.closeCreateDrawer}
            onSuccess={() => {
              listReloadRef.current();
              ctx.closeCreateDrawer();
            }}
          />
        )}
        renderToolbarExtraRight={(ctx) => (
          <Button
            disabled={!ctx.clusterId}
            onClick={() => {
              if (!ctx.clusterId) return;
              Modal.confirm({
                title: "重启 Ingress-Nginx Controller Pods",
                content: "将删除 ingress-nginx controller Pods 以触发自动重建，用于刷新默认证书等运行态资源。确认继续吗？",
                okText: "确认重启",
                cancelText: "取消",
                onOk: async () => {
                  const res = await restartIngressNginxPods(ctx.clusterId!);
                  message.success(`已删除 ${res.deleted_count} 个 Pod`);
                  await ctx.reload();
                },
              });
            }}
          >
            重启 Ingress-Nginx
          </Button>
        )}
        createTemplate={({ namespace }) => `apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: demo-ingress
  namespace: ${namespace ?? "default"}
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
spec:
  ingressClassName: nginx
  tls:
    - hosts:
        - demo.local
      secretName: demo-local-tls
  rules:
    - host: demo.local
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: demo-service
                port:
                  name: http
`}
      />

      {viewer}
    </>
  );
}
