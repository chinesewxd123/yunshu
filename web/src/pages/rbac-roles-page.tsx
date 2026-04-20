import { Tag } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useRef } from "react";
import { RbacRoleFormCreateDrawer } from "../components/k8s/k8s-resource-form-drawers";
import { YamlCrudPage } from "../components/k8s/yaml-crud-page";
import { listNamespaces as listClusterNamespaces } from "../services/clusters";
import { applyRbac, deleteRbac, getRbacDetail, listRoles, type RbacDetail, type RbacRoleItem } from "../services/rbac";

type Item = RbacRoleItem;
type Detail = RbacDetail;

export function RbacRolesPage() {
  const listReloadRef = useRef<() => void>(() => {});

  const columns: ColumnsType<Item> = [
    { title: "名称", dataIndex: "name", width: 260 },
    { title: "规则数", dataIndex: "rules", width: 100, render: (v: number) => <Tag color="blue">{v}</Tag> },
    { title: "创建时间", dataIndex: "creation_time", width: 180, fixed: "right" },
  ];

  return (
    <>
    <YamlCrudPage<Item, Detail>
      title="RBAC - Role"
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
        <RbacRoleFormCreateDrawer
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
        list: async ({ clusterId, namespace, keyword }) => (await listRoles(clusterId, namespace ?? "default", keyword)).list,
        detail: async ({ clusterId, namespace, name }) =>
          await getRbacDetail({ cluster_id: clusterId, kind: "Role", namespace: namespace ?? "default", name }),
        apply: async ({ clusterId, manifest }) => await applyRbac(clusterId, manifest),
        remove: async ({ clusterId, namespace, name }) =>
          await deleteRbac({ cluster_id: clusterId, kind: "Role", namespace: namespace ?? "default", name }),
      }}
      createTemplate={({ namespace }) => `apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: demo-role
  namespace: ${namespace || "default"}
rules:
  - apiGroups: [""]
    resources: ["pods"]
    verbs: ["get", "list"]
`}
    />
    </>
  );
}

