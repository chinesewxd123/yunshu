import { Space, Tag, Typography } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useRef } from "react";
import { ServiceAccountFormCreateDrawer } from "../components/k8s/k8s-resource-form-drawers";
import { YamlCrudPage } from "../components/k8s/yaml-crud-page";
import { listNamespaces as listClusterNamespaces } from "../services/clusters";
import {
  applyServiceAccount,
  deleteServiceAccount,
  getServiceAccountDetail,
  listServiceAccounts,
  type ServiceAccountDetail,
  type ServiceAccountItem,
} from "../services/serviceaccounts";

export function ServiceaccountsPage() {
  const listReloadRef = useRef<() => void>(() => {});
  const columns: ColumnsType<ServiceAccountItem> = [
    { title: "名称", dataIndex: "name" },
    { title: "Secrets", dataIndex: "secrets_count", width: 110 },
    { title: "ImagePullSecrets", dataIndex: "image_pull_secrets_count", width: 150 },
    { title: "创建时间", dataIndex: "creation_time", width: 180, fixed: "right" },
  ];

  return (
    <YamlCrudPage<ServiceAccountItem, ServiceAccountDetail>
      title="ServiceAccount 管理"
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
        <ServiceAccountFormCreateDrawer
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
        list: async ({ clusterId, namespace, keyword }) => await listServiceAccounts(clusterId, namespace ?? "default", keyword),
        detail: async ({ clusterId, namespace, name }) => await getServiceAccountDetail(clusterId, namespace ?? "default", name),
        apply: async ({ clusterId, manifest }) => await applyServiceAccount(clusterId, manifest),
        remove: async ({ clusterId, namespace, name }) => await deleteServiceAccount(clusterId, namespace ?? "default", name),
      }}
      detailExtra={(detail) => (
        <Space direction="vertical" style={{ width: "100%" }} size={10}>
          <Space wrap>
            <Typography.Text type="secondary">Secrets:</Typography.Text>
            {detail.secrets?.length ? detail.secrets.map((it) => <Tag key={it}>{it}</Tag>) : <span className="inline-muted">-</span>}
          </Space>
          <Space wrap>
            <Typography.Text type="secondary">ImagePullSecrets:</Typography.Text>
            {detail.image_pull_secrets?.length ? (
              detail.image_pull_secrets.map((it) => <Tag key={it} color="blue">{it}</Tag>)
            ) : (
              <span className="inline-muted">-</span>
            )}
          </Space>
          <Space direction="vertical" size={4}>
            <Typography.Text type="secondary">RoleBindings:</Typography.Text>
            <Space wrap>
              {detail.role_bindings?.length ? (
                detail.role_bindings.map((it) => (
                  <Tag key={`${it.namespace}/${it.name}`}>
                    {(it.namespace ? `${it.namespace}/` : "") + it.name} -> {it.role_ref}
                  </Tag>
                ))
              ) : (
                <span className="inline-muted">-</span>
              )}
            </Space>
          </Space>
          <Space direction="vertical" size={4}>
            <Typography.Text type="secondary">ClusterRoleBindings:</Typography.Text>
            <Space wrap>
              {detail.cluster_role_bindings?.length ? (
                detail.cluster_role_bindings.map((it) => (
                  <Tag key={it.name} color="purple">
                    {it.name} -> {it.role_ref}
                  </Tag>
                ))
              ) : (
                <span className="inline-muted">-</span>
              )}
            </Space>
          </Space>
        </Space>
      )}
      createTemplate={({ namespace }) => `apiVersion: v1
kind: ServiceAccount
metadata:
  name: demo-sa
  namespace: ${namespace ?? "default"}
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: demo-sa-view
  namespace: ${namespace ?? "default"}
subjects:
  - kind: ServiceAccount
    name: demo-sa
    namespace: ${namespace ?? "default"}
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: view
`}
    />
  );
}
