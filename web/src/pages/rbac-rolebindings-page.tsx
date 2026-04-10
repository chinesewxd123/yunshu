import { Space, Tag, Typography } from "antd";
import type { ColumnsType } from "antd/es/table";
import { YamlCrudPage } from "../components/k8s/yaml-crud-page";
import { listNamespaces as listClusterNamespaces } from "../services/clusters";
import { applyRbac, deleteRbac, getRbacDetail, listRoleBindings, type RbacDetail, type RbacRoleBindingItem } from "../services/rbac";

type Item = RbacRoleBindingItem;
type Detail = RbacDetail;

export function RbacRoleBindingsPage() {
  const columns: ColumnsType<Item> = [
    { title: "名称", dataIndex: "name", width: 260 },
    { title: "RoleRef", dataIndex: "role_ref", width: 240, render: (v: string) => <Tag>{v || "-"}</Tag> },
    {
      title: "Subjects",
      dataIndex: "subjects",
      render: (v?: string[]) =>
        v?.length ? (
          <Space wrap size={[6, 6]}>
            {v.slice(0, 8).map((s) => (
              <Tag key={s}>{s}</Tag>
            ))}
            {v.length > 8 ? <Typography.Text type="secondary">+{v.length - 8}</Typography.Text> : null}
          </Space>
        ) : (
          <span className="inline-muted">-</span>
        ),
    },
    { title: "创建时间", dataIndex: "creation_time", width: 180, fixed: "right" },
  ];

  return (
    <YamlCrudPage<Item, Detail>
      title="RBAC - RoleBinding"
      needNamespace
      onLoadNamespaces={async (cid) => {
        const res = await listClusterNamespaces(cid);
        return (res.list ?? []).map((n) => ({ label: n.name, value: n.name }));
      }}
      columns={columns}
      api={{
        list: async ({ clusterId, namespace, keyword }) => (await listRoleBindings(clusterId, namespace ?? "default", keyword)).list,
        detail: async ({ clusterId, namespace, name }) =>
          await getRbacDetail({ cluster_id: clusterId, kind: "RoleBinding", namespace: namespace ?? "default", name }),
        apply: async ({ clusterId, manifest }) => await applyRbac(clusterId, manifest),
        remove: async ({ clusterId, namespace, name }) =>
          await deleteRbac({ cluster_id: clusterId, kind: "RoleBinding", namespace: namespace ?? "default", name }),
      }}
      createTemplate={({ namespace }) => `apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: demo-rolebinding
  namespace: ${namespace || "default"}
subjects:
  - kind: User
    name: demo-user
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: demo-role
`}
    />
  );
}

