import { FileTextOutlined, TagsOutlined } from "@ant-design/icons";
import { Button, Card, Col, Form, Input, InputNumber, Row, Select, Space, Tag, message } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useRef, useState } from "react";
import YAML from "yaml";
import { useKeyValueViewer } from "../components/k8s/key-value-viewer";
import { WorkloadFormModal } from "../components/k8s/workload-forms";
import { LabelsFormList } from "../components/k8s/service-storage-forms";
import { YamlCrudPage } from "../components/k8s/yaml-crud-page";
import { listNamespaces as listClusterNamespaces } from "../services/clusters";
import {
  applyNetworkPolicy,
  deleteNetworkPolicy,
  getNetworkPolicyDetail,
  listNetworkPolicies,
  type NetworkPolicyDetail,
  type NetworkPolicyItem,
} from "../services/network-policies";

type KVPair = { key?: string; value?: string };
type NetworkPolicyPortForm = {
  protocol?: "TCP" | "UDP" | "SCTP";
  port?: string;
  endPort?: number;
};
type NetworkPolicyPeerForm = {
  pod_selector_pairs?: KVPair[];
  namespace_selector_pairs?: KVPair[];
  ip_cidr?: string;
  ip_except_csv?: string;
};
type NetworkPolicyRuleForm = {
  peers?: NetworkPolicyPeerForm[];
  ports?: NetworkPolicyPortForm[];
};
type NetworkPolicyFormValues = {
  name: string;
  namespace: string;
  policy_types?: Array<"Ingress" | "Egress">;
  pod_selector_pairs?: KVPair[];
  ingress?: NetworkPolicyRuleForm[];
  egress?: NetworkPolicyRuleForm[];
};

function pairsToMap(pairs?: KVPair[]): Record<string, string> | undefined {
  const out: Record<string, string> = {};
  for (const p of pairs ?? []) {
    const k = String(p?.key ?? "").trim();
    if (!k) continue;
    out[k] = String(p?.value ?? "");
  }
  return Object.keys(out).length ? out : undefined;
}

function mapToPairs(m?: Record<string, string>): KVPair[] {
  const out = Object.entries(m ?? {}).map(([key, value]) => ({ key, value: String(value ?? "") }));
  return out.length ? out : [{ key: "", value: "" }];
}

function parsePortValue(v?: string): string | number | undefined {
  const s = String(v ?? "").trim();
  if (!s) return undefined;
  if (/^\d+$/.test(s)) {
    const n = Number(s);
    if (Number.isFinite(n) && n > 0) return n;
  }
  return s;
}

function buildNetworkPolicyYaml(v: NetworkPolicyFormValues): string {
  const ingress = (v.ingress ?? [])
    .map((rule) => {
      const from = (rule.peers ?? [])
        .map((peer) => {
          const podSel = pairsToMap(peer.pod_selector_pairs);
          const nsSel = pairsToMap(peer.namespace_selector_pairs);
          const ipCIDR = String(peer.ip_cidr ?? "").trim();
          const ipExcept = String(peer.ip_except_csv ?? "")
            .split(",")
            .map((x) => x.trim())
            .filter(Boolean);
          if (!podSel && !nsSel && !ipCIDR) return undefined;
          return {
            ...(podSel ? { podSelector: { matchLabels: podSel } } : {}),
            ...(nsSel ? { namespaceSelector: { matchLabels: nsSel } } : {}),
            ...(ipCIDR ? { ipBlock: { cidr: ipCIDR, ...(ipExcept.length ? { except: ipExcept } : {}) } } : {}),
          };
        })
        .filter(Boolean);
      const ports = (rule.ports ?? [])
        .map((p) => {
          const port = parsePortValue(p.port);
          if (!port) return undefined;
          return {
            ...(p.protocol ? { protocol: p.protocol } : {}),
            port,
            ...(p.endPort && p.endPort > 0 ? { endPort: p.endPort } : {}),
          };
        })
        .filter(Boolean);
      if (!from.length && !ports.length) return undefined;
      return {
        ...(from.length ? { from } : {}),
        ...(ports.length ? { ports } : {}),
      };
    })
    .filter(Boolean);

  const egress = (v.egress ?? [])
    .map((rule) => {
      const to = (rule.peers ?? [])
        .map((peer) => {
          const podSel = pairsToMap(peer.pod_selector_pairs);
          const nsSel = pairsToMap(peer.namespace_selector_pairs);
          const ipCIDR = String(peer.ip_cidr ?? "").trim();
          const ipExcept = String(peer.ip_except_csv ?? "")
            .split(",")
            .map((x) => x.trim())
            .filter(Boolean);
          if (!podSel && !nsSel && !ipCIDR) return undefined;
          return {
            ...(podSel ? { podSelector: { matchLabels: podSel } } : {}),
            ...(nsSel ? { namespaceSelector: { matchLabels: nsSel } } : {}),
            ...(ipCIDR ? { ipBlock: { cidr: ipCIDR, ...(ipExcept.length ? { except: ipExcept } : {}) } } : {}),
          };
        })
        .filter(Boolean);
      const ports = (rule.ports ?? [])
        .map((p) => {
          const port = parsePortValue(p.port);
          if (!port) return undefined;
          return {
            ...(p.protocol ? { protocol: p.protocol } : {}),
            port,
            ...(p.endPort && p.endPort > 0 ? { endPort: p.endPort } : {}),
          };
        })
        .filter(Boolean);
      if (!to.length && !ports.length) return undefined;
      return {
        ...(to.length ? { to } : {}),
        ...(ports.length ? { ports } : {}),
      };
    })
    .filter(Boolean);

  const podSelector = pairsToMap(v.pod_selector_pairs) ?? {};
  const policyTypes = (v.policy_types ?? []).filter(Boolean);
  const obj: any = {
    apiVersion: "networking.k8s.io/v1",
    kind: "NetworkPolicy",
    metadata: {
      name: String(v.name ?? "").trim(),
      namespace: String(v.namespace ?? "default").trim() || "default",
    },
    spec: {
      podSelector: { matchLabels: podSelector },
      ...(policyTypes.length ? { policyTypes } : {}),
      ...(ingress.length ? { ingress } : {}),
      ...(egress.length ? { egress } : {}),
    },
  };
  return YAML.stringify(obj);
}

function networkPolicyYamlToForm(text: string): NetworkPolicyFormValues | null {
  try {
    const obj: any = YAML.parse(text);
    const ingress = Array.isArray(obj?.spec?.ingress)
      ? obj.spec.ingress.map((rule: any) => ({
          peers: (Array.isArray(rule?.from) ? rule.from : []).map((peer: any) => ({
            pod_selector_pairs: mapToPairs(peer?.podSelector?.matchLabels),
            namespace_selector_pairs: mapToPairs(peer?.namespaceSelector?.matchLabels),
            ip_cidr: peer?.ipBlock?.cidr ? String(peer.ipBlock.cidr) : undefined,
            ip_except_csv: Array.isArray(peer?.ipBlock?.except) ? peer.ipBlock.except.join(",") : undefined,
          })),
          ports: (Array.isArray(rule?.ports) ? rule.ports : []).map((port: any) => ({
            protocol: port?.protocol,
            port: port?.port === undefined || port?.port === null ? undefined : String(port.port),
            endPort: typeof port?.endPort === "number" ? port.endPort : undefined,
          })),
        }))
      : [];

    const egress = Array.isArray(obj?.spec?.egress)
      ? obj.spec.egress.map((rule: any) => ({
          peers: (Array.isArray(rule?.to) ? rule.to : []).map((peer: any) => ({
            pod_selector_pairs: mapToPairs(peer?.podSelector?.matchLabels),
            namespace_selector_pairs: mapToPairs(peer?.namespaceSelector?.matchLabels),
            ip_cidr: peer?.ipBlock?.cidr ? String(peer.ipBlock.cidr) : undefined,
            ip_except_csv: Array.isArray(peer?.ipBlock?.except) ? peer.ipBlock.except.join(",") : undefined,
          })),
          ports: (Array.isArray(rule?.ports) ? rule.ports : []).map((port: any) => ({
            protocol: port?.protocol,
            port: port?.port === undefined || port?.port === null ? undefined : String(port.port),
            endPort: typeof port?.endPort === "number" ? port.endPort : undefined,
          })),
        }))
      : [];

    return {
      name: String(obj?.metadata?.name ?? ""),
      namespace: String(obj?.metadata?.namespace ?? "default"),
      policy_types: Array.isArray(obj?.spec?.policyTypes) ? obj.spec.policyTypes : undefined,
      pod_selector_pairs: mapToPairs(obj?.spec?.podSelector?.matchLabels),
      ingress: ingress.length ? ingress : [{ peers: [{ pod_selector_pairs: [{ key: "", value: "" }] }], ports: [{ protocol: "TCP", port: "80" }] }],
      egress: egress.length ? egress : [{ peers: [{ ip_cidr: "0.0.0.0/0" }], ports: [{ protocol: "TCP", port: "443" }] }],
    };
  } catch {
    return null;
  }
}

function RuleFormList({ name, peerLabel }: { name: "ingress" | "egress"; peerLabel: "from" | "to" }) {
  return (
    <Form.List name={name}>
      {(ruleFields, { add: addRule, remove: removeRule }) => (
        <Space direction="vertical" style={{ width: "100%" }}>
          {ruleFields.map((ruleField) => (
            <Card
              key={ruleField.key}
              size="small"
              title={`${name === "ingress" ? "Ingress" : "Egress"} 规则`}
              extra={<Button onClick={() => removeRule(ruleField.name)}>删除规则</Button>}
            >
              <Form.List name={[ruleField.name, "peers"]}>
                {(peerFields, { add: addPeer, remove: removePeer }) => (
                  <Space direction="vertical" style={{ width: "100%" }}>
                    {peerFields.map((peerField) => (
                      <Card key={peerField.key} size="small" title={`${peerLabel} Peer`} extra={<Button onClick={() => removePeer(peerField.name)}>删除 Peer</Button>}>
                        <Row gutter={12}>
                          <Col span={12}>
                            <Form.Item label="IPBlock CIDR" name={[peerField.name, "ip_cidr"]}>
                              <Input placeholder="10.0.0.0/24" />
                            </Form.Item>
                          </Col>
                          <Col span={12}>
                            <Form.Item label="IPBlock Except（逗号分隔）" name={[peerField.name, "ip_except_csv"]}>
                              <Input placeholder="10.0.0.10/32,10.0.0.11/32" />
                            </Form.Item>
                          </Col>
                        </Row>
                        <Form.Item label="Pod Selector">
                          <LabelsFormList name={[ruleField.name, "peers", peerField.name, "pod_selector_pairs"] as any} addLabel="新增 PodSelector" />
                        </Form.Item>
                        <Form.Item label="Namespace Selector">
                          <LabelsFormList name={[ruleField.name, "peers", peerField.name, "namespace_selector_pairs"] as any} addLabel="新增 NamespaceSelector" />
                        </Form.Item>
                      </Card>
                    ))}
                    <Button onClick={() => addPeer({ pod_selector_pairs: [{ key: "", value: "" }], namespace_selector_pairs: [{ key: "", value: "" }] })}>新增 Peer</Button>
                  </Space>
                )}
              </Form.List>
              <Form.List name={[ruleField.name, "ports"]}>
                {(portFields, { add: addPort, remove: removePort }) => (
                  <Space direction="vertical" style={{ width: "100%" }}>
                    {portFields.map((portField) => (
                      <Space key={portField.key} align="start" wrap>
                        <Form.Item label="协议" name={[portField.name, "protocol"]} style={{ width: 120 }}>
                          <Select allowClear options={[{ label: "TCP", value: "TCP" }, { label: "UDP", value: "UDP" }, { label: "SCTP", value: "SCTP" }]} />
                        </Form.Item>
                        <Form.Item label="端口（数字或名称）" name={[portField.name, "port"]} style={{ width: 200 }}>
                          <Input placeholder="80 或 http" />
                        </Form.Item>
                        <Form.Item label="endPort（可选）" name={[portField.name, "endPort"]} style={{ width: 160 }}>
                          <InputNumber min={1} max={65535} style={{ width: "100%" }} />
                        </Form.Item>
                        <Button onClick={() => removePort(portField.name)} style={{ marginTop: 30 }}>
                          删除端口
                        </Button>
                      </Space>
                    ))}
                    <Button onClick={() => addPort({ protocol: "TCP", port: "80" })}>新增端口规则</Button>
                  </Space>
                )}
              </Form.List>
            </Card>
          ))}
          <Button onClick={() => addRule({ peers: [{ pod_selector_pairs: [{ key: "", value: "" }], namespace_selector_pairs: [{ key: "", value: "" }] }], ports: [{ protocol: "TCP", port: "80" }] })}>
            新增 {name === "ingress" ? "Ingress" : "Egress"} 规则
          </Button>
        </Space>
      )}
    </Form.List>
  );
}

export function NetworkPoliciesPage() {
  const listReloadRef = useRef<() => void>(() => {});
  const { renderKVIcon, viewer } = useKeyValueViewer();
  const [formOpen, setFormOpen] = useState(false);
  const [formLoading, setFormLoading] = useState(false);
  const [formMode, setFormMode] = useState<"create" | "edit">("create");
  const [formCtx, setFormCtx] = useState<{ clusterId: number; namespace: string; name?: string } | null>(null);
  const [form] = Form.useForm<NetworkPolicyFormValues>();

  const columns: ColumnsType<NetworkPolicyItem> = [
    { title: "命名空间", dataIndex: "namespace", width: 120 },
    { title: "名称", dataIndex: "name", width: 220 },
    { title: "策略类型", dataIndex: "policy_types", width: 150 },
    { title: "Pod选择器数", dataIndex: "pod_selector_count", width: 110 },
    { title: "Ingress规则数", dataIndex: "ingress_rule_count", width: 120 },
    { title: "Egress规则数", dataIndex: "egress_rule_count", width: 120 },
    { title: "标签", key: "labels", width: 70, align: "center", render: (_, r) => renderKVIcon("标签", <TagsOutlined />, r.labels) },
    { title: "注解", key: "annotations", width: 70, align: "center", render: (_, r) => renderKVIcon("注解", <FileTextOutlined />, r.annotations) },
    { title: "存在时长", dataIndex: "age", width: 90, fixed: "right" },
    { title: "创建时间", dataIndex: "creation_time", width: 170, fixed: "right" },
  ];

  return (
    <>
      <YamlCrudPage<NetworkPolicyItem, NetworkPolicyDetail>
        title="网络策略管理"
        needNamespace
        onLoadNamespaces={async (cid) => {
          const res = await listClusterNamespaces(cid);
          return (res.list ?? []).map((n) => ({ label: n.name, value: n.name }));
        }}
        showEditButton={false}
        columns={columns}
        api={{
          list: async ({ clusterId, namespace, keyword }) => await listNetworkPolicies(clusterId, namespace ?? "default", keyword),
          detail: async ({ clusterId, namespace, name }) => await getNetworkPolicyDetail(clusterId, namespace ?? "default", name),
          apply: async ({ clusterId, manifest }) => await applyNetworkPolicy(clusterId, manifest),
          remove: async ({ clusterId, namespace, name }) => await deleteNetworkPolicy(clusterId, namespace ?? "default", name),
        }}
        onToolbarReady={(ctx) => {
          listReloadRef.current = ctx.reload;
        }}
        onCreateDrawerOpen={(ctx) => {
          if (!ctx.clusterId) return;
          setFormMode("create");
          setFormCtx({ clusterId: ctx.clusterId, namespace: ctx.namespace ?? "default" });
          form.setFieldsValue({
            name: "",
            namespace: ctx.namespace ?? "default",
            policy_types: ["Ingress", "Egress"],
            pod_selector_pairs: [{ key: "app", value: "" }],
            ingress: [{ peers: [{ namespace_selector_pairs: [{ key: "kubernetes.io/metadata.name", value: ctx.namespace ?? "default" }] }], ports: [{ protocol: "TCP", port: "80" }] }],
            egress: [{ peers: [{ ip_cidr: "0.0.0.0/0" }], ports: [{ protocol: "TCP", port: "443" }] }],
          });
        }}
        renderCreateFormTab={(drawerCtx) => (
          <WorkloadFormModal<NetworkPolicyFormValues>
            embedded
            title="网络策略表单创建"
            open={false}
            loading={formLoading}
            form={form}
            onCancel={drawerCtx.closeCreateDrawer}
            onSubmit={(values) => {
              const cid = drawerCtx.clusterId;
              if (!cid) return;
              setFormLoading(true);
              void (async () => {
                try {
                  await applyNetworkPolicy(cid, buildNetworkPolicyYaml(values));
                  message.success("NetworkPolicy 已应用");
                  drawerCtx.closeCreateDrawer();
                  listReloadRef.current();
                } finally {
                  setFormLoading(false);
                }
              })();
            }}
          >
            <Row gutter={16}>
              <Col span={14}>
                <Form.Item name="name" label="名称" rules={[{ required: true, message: "请输入名称" }]}>
                  <Input />
                </Form.Item>
              </Col>
              <Col span={10}>
                <Form.Item name="namespace" label="命名空间" rules={[{ required: true, message: "请输入命名空间" }]}>
                  <Input />
                </Form.Item>
              </Col>
            </Row>
            <Form.Item name="policy_types" label="PolicyTypes">
              <Select mode="multiple" allowClear options={[{ label: "Ingress", value: "Ingress" }, { label: "Egress", value: "Egress" }]} />
            </Form.Item>
            <Form.Item label="Pod Selector（受控 Pod）">
              <LabelsFormList name={"pod_selector_pairs" as any} addLabel="新增 PodSelector" />
            </Form.Item>
            <Card size="small" title="Ingress 规则" style={{ marginBottom: 12 }}>
              <RuleFormList name="ingress" peerLabel="from" />
            </Card>
            <Card size="small" title="Egress 规则">
              <RuleFormList name="egress" peerLabel="to" />
            </Card>
          </WorkloadFormModal>
        )}
        extraRowActions={(record, ctx) => (
          <Button
            type="link"
            onClick={() => {
              setFormMode("edit");
              setFormCtx({ clusterId: ctx.clusterId, namespace: ctx.namespace ?? "default", name: record.name });
              setFormOpen(true);
              setFormLoading(true);
              void (async () => {
                try {
                  const d = await getNetworkPolicyDetail(ctx.clusterId, ctx.namespace ?? "default", record.name);
                  const fv = networkPolicyYamlToForm(d.yaml ?? "");
                  if (fv) {
                    form.setFieldsValue(fv);
                  }
                } finally {
                  setFormLoading(false);
                }
              })();
            }}
          >
            表单编辑
          </Button>
        )}
        createTemplate={({ namespace }) => `apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: demo-network-policy
  namespace: ${namespace ?? "default"}
spec:
  podSelector:
    matchLabels:
      app: demo
  policyTypes:
    - Ingress
    - Egress
  ingress:
    - from:
        - namespaceSelector:
            matchLabels:
              kubernetes.io/metadata.name: ${namespace ?? "default"}
      ports:
        - protocol: TCP
          port: 80
  egress:
    - to:
        - ipBlock:
            cidr: 0.0.0.0/0
      ports:
        - protocol: TCP
          port: 443
`}
      />
      <WorkloadFormModal<NetworkPolicyFormValues>
        title="网络策略表单编辑"
        open={formOpen && formMode === "edit"}
        loading={formLoading}
        form={form}
        onCancel={() => setFormOpen(false)}
        onSubmit={(values) => {
          if (!formCtx) return;
          setFormLoading(true);
          void (async () => {
            try {
              await applyNetworkPolicy(formCtx.clusterId, buildNetworkPolicyYaml(values));
              message.success("NetworkPolicy 已应用");
              setFormOpen(false);
              listReloadRef.current();
            } finally {
              setFormLoading(false);
            }
          })();
        }}
      >
        <Row gutter={16}>
          <Col span={14}>
            <Form.Item name="name" label="名称" rules={[{ required: true, message: "请输入名称" }]}>
              <Input />
            </Form.Item>
          </Col>
          <Col span={10}>
            <Form.Item name="namespace" label="命名空间" rules={[{ required: true, message: "请输入命名空间" }]}>
              <Input />
            </Form.Item>
          </Col>
        </Row>
        <Form.Item name="policy_types" label="PolicyTypes">
          <Select mode="multiple" allowClear options={[{ label: "Ingress", value: "Ingress" }, { label: "Egress", value: "Egress" }]} />
        </Form.Item>
        <Form.Item label="Pod Selector（受控 Pod）">
          <LabelsFormList name={"pod_selector_pairs" as any} addLabel="新增 PodSelector" />
        </Form.Item>
        <Card size="small" title="Ingress 规则" style={{ marginBottom: 12 }}>
          <RuleFormList name="ingress" peerLabel="from" />
        </Card>
        <Card size="small" title="Egress 规则">
          <RuleFormList name="egress" peerLabel="to" />
        </Card>
      </WorkloadFormModal>
      {viewer}
    </>
  );
}
