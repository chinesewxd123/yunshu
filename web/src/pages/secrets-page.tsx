import { Alert, Typography } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useRef } from "react";
import { SecretFormCreateDrawer } from "../components/k8s/k8s-resource-form-drawers";
import { YamlCrudPage } from "../components/k8s/yaml-crud-page";
import { listNamespaces as listClusterNamespaces } from "../services/clusters";
import { applySecret, deleteSecret, getSecretDetail, listSecrets, type ConfigDetail, type SecretItem } from "../services/configs";

export function SecretsPage() {
  const listReloadRef = useRef<() => void>(() => {});

  const columns: ColumnsType<SecretItem> = [
    { title: "名称", dataIndex: "name" },
    { title: "类型", dataIndex: "type", width: 180 },
    { title: "键数量", dataIndex: "data_count", width: 120 },
    { title: "创建时间", dataIndex: "creation_time", width: 180, fixed: "right" },
  ];

  return (
    <>
    <YamlCrudPage<SecretItem, ConfigDetail>
      title="Secret 管理"
      needNamespace
      onLoadNamespaces={async (cid) => {
        const res = await listClusterNamespaces(cid);
        return (res.list ?? []).map((n) => ({ label: n.name, value: n.name }));
      }}
      columns={columns}
      onToolbarReady={(ctx) => {
        listReloadRef.current = ctx.reload;
      }}
      renderCreateFormTab={(ctx) => (
        <SecretFormCreateDrawer
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
      api={{
        list: async ({ clusterId, namespace, keyword }) => await listSecrets(clusterId, namespace ?? "default", keyword),
        detail: async ({ clusterId, namespace, name }) => await getSecretDetail(clusterId, namespace ?? "default", name),
        apply: async ({ clusterId, manifest }) => await applySecret(clusterId, manifest),
        remove: async ({ clusterId, namespace, name }) => await deleteSecret(clusterId, namespace ?? "default", name),
      }}
      createTemplate={({ namespace }) => `apiVersion: v1
kind: Secret
metadata:
  name: demo-secret
  namespace: ${namespace ?? "default"}
type: Opaque
stringData:
  username: admin
  password: Admin@123
`}
      detailExtra={(d) => (
        <div>
          <Alert
            type="warning"
            showIcon
            message="注意"
            description="Secret 中的 data 可能包含敏感信息。下方 YAML 会包含 base64 内容；decoded_data 仅用于调试查看。"
          />
          {d.decoded_data ? (
            <div style={{ marginTop: 12 }}>
              <Typography.Title level={5} style={{ marginTop: 0 }}>
                decoded_data（仅供查看）
              </Typography.Title>
              <Typography.Paragraph style={{ whiteSpace: "pre-wrap" }}>
                {Object.entries(d.decoded_data)
                  .map(([k, v]) => `${k}: ${v}`)
                  .join("\n")}
              </Typography.Paragraph>
            </div>
          ) : null}
        </div>
      )}
    />
    </>
  );
}

