import { Tag } from "antd";
import type { ColumnsType } from "antd/es/table";
import { YamlCrudPage } from "../components/k8s/yaml-crud-page";
import { applyRbac, deleteRbac, getRbacDetail, listClusterRoles, type RbacClusterRoleItem, type RbacDetail } from "../services/rbac";

type Item = RbacClusterRoleItem;
type Detail = RbacDetail;

export function RbacClusterRolesPage() {
  const columns: ColumnsType<Item> = [
    { title: "名称", dataIndex: "name", width: 280 },
    { title: "规则数", dataIndex: "rules", width: 100, render: (v: number) => <Tag color="blue">{v}</Tag> },
    { title: "创建时间", dataIndex: "creation_time", width: 180, fixed: "right" },
  ];

  return (
    <YamlCrudPage<Item, Detail>
      title="RBAC - ClusterRole"
      needNamespace={false}
      columns={columns}
      api={{
        list: async ({ clusterId, keyword }) => (await listClusterRoles(clusterId, keyword)).list,
        detail: async ({ clusterId, name }) => await getRbacDetail({ cluster_id: clusterId, kind: "ClusterRole", name }),
        apply: async ({ clusterId, manifest }) => await applyRbac(clusterId, manifest),
        remove: async ({ clusterId, name }) =>
          await deleteRbac({ cluster_id: clusterId, kind: "ClusterRole", name }),
      }}
      createTemplate={() => `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: demo-clusterrole
rules:
  - apiGroups: [""]
    resources: ["nodes"]
    verbs: ["get", "list"]
`}
    />
  );
}

