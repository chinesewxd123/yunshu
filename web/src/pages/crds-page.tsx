import { Tag } from "antd";
import type { ColumnsType } from "antd/es/table";
import { YamlCrudPage } from "../components/k8s/yaml-crud-page";
import { applyCrd, deleteCrd, getCrdDetail, listCrds, type CrdDetail, type CrdItem } from "../services/crds";

export function CrdsPage() {
  const columns: ColumnsType<CrdItem> = [
    { title: "名称", dataIndex: "name", width: 280 },
    { title: "Group", dataIndex: "group", width: 220 },
    { title: "Kind", dataIndex: "kind", width: 160 },
    { title: "Plural", dataIndex: "plural", width: 160 },
    { title: "作用域", dataIndex: "scope", width: 120 },
    { title: "版本", dataIndex: "current_version", width: 120 },
    {
      title: "状态",
      dataIndex: "established",
      width: 100,
      render: (v: boolean) => (v ? <Tag color="green">已建立</Tag> : <Tag color="orange">未就绪</Tag>),
    },
    { title: "创建时间", dataIndex: "creation_time", width: 180, fixed: "right" },
  ];

  return (
    <YamlCrudPage<CrdItem, CrdDetail>
      title="CRD 管理"
      columns={columns}
      api={{
        list: async ({ clusterId, keyword }) => await listCrds(clusterId, keyword),
        detail: async ({ clusterId, name }) => await getCrdDetail(clusterId, name),
        apply: async ({ clusterId, manifest }) => await applyCrd(clusterId, manifest),
        remove: async ({ clusterId, name }) => await deleteCrd(clusterId, name),
      }}
      createTemplate={() => `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: widgets.demo.example.com
spec:
  group: demo.example.com
  scope: Namespaced
  names:
    plural: widgets
    singular: widget
    kind: Widget
    shortNames:
      - wd
  versions:
    - name: v1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              properties:
                size:
                  type: string
`}
    />
  );
}
