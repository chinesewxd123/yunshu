import { Space, Tag, Typography } from "antd";
import type { ColumnsType } from "antd/es/table";
import { YamlCrudPage } from "../components/k8s/yaml-crud-page";
import { applyRbac, deleteRbac, getRbacDetail, listClusterRoleBindings, type RbacClusterRoleBindingItem, type RbacDetail } from "../services/rbac";

type Item = RbacClusterRoleBindingItem;
type Detail = RbacDetail;

export function RbacClusterRoleBindingsPage() {
  const columns: ColumnsType<Item> = [
    { title: "名称", dataIndex: "name", width: 280 },
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
      title="RBAC - ClusterRoleBinding"
      needNamespace={false}
      columns={columns}
      api={{
        list: async ({ clusterId, keyword }) => (await listClusterRoleBindings(clusterId, keyword)).list,
        detail: async ({ clusterId, name }) => await getRbacDetail({ cluster_id: clusterId, kind: "ClusterRoleBinding", name }),
        apply: async ({ clusterId, manifest }) => await applyRbac(clusterId, manifest),
        remove: async ({ clusterId, name }) =>
          await deleteRbac({ cluster_id: clusterId, kind: "ClusterRoleBinding", name }),
      }}
      createTemplate={() => `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: demo-clusterrolebinding
subjects:
  - kind: User
    name: demo-user
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: demo-clusterrole
`}
    />
  );
}

