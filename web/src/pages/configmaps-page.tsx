import type { ColumnsType } from "antd/es/table";
import { YamlCrudPage } from "../components/k8s/yaml-crud-page";
import { listNamespaces as listClusterNamespaces } from "../services/clusters";
import { applyConfigMap, deleteConfigMap, getConfigMapDetail, listConfigMaps, type ConfigDetail, type ConfigMapItem } from "../services/configs";

export function ConfigmapsPage() {
  const columns: ColumnsType<ConfigMapItem> = [
    { title: "名称", dataIndex: "name" },
    { title: "键数量", dataIndex: "data_count", width: 120 },
    { title: "创建时间", dataIndex: "creation_time", width: 180, fixed: "right" },
  ];

  return (
    <YamlCrudPage<ConfigMapItem, ConfigDetail>
      title="ConfigMap 管理"
      needNamespace
      onLoadNamespaces={async (cid) => {
        const res = await listClusterNamespaces(cid);
        return (res.list ?? []).map((n) => ({ label: n.name, value: n.name }));
      }}
      columns={columns}
      api={{
        list: async ({ clusterId, namespace, keyword }) => await listConfigMaps(clusterId, namespace ?? "default", keyword),
        detail: async ({ clusterId, namespace, name }) => await getConfigMapDetail(clusterId, namespace ?? "default", name),
        apply: async ({ clusterId, manifest }) => await applyConfigMap(clusterId, manifest),
        remove: async ({ clusterId, namespace, name }) => await deleteConfigMap(clusterId, namespace ?? "default", name),
      }}
      createTemplate={({ namespace }) => `apiVersion: v1
kind: ConfigMap
metadata:
  name: demo-config
  namespace: ${namespace ?? "default"}
data:
  app.env: "prod"
  feature.flag: "true"
`}
    />
  );
}

