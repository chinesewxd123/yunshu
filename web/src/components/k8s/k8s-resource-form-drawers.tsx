import { Button, Drawer, Form, Input, InputNumber, Select, Space, Switch, Typography, message } from "antd";
import type { FormInstance } from "antd/es/form";
import { useState } from "react";
import YAML from "yaml";
import { applyNamespace } from "../../services/namespaces";
import { applyConfigMap, applySecret } from "../../services/configs";
import { applyIngress } from "../../services/ingresses";
import { applyRbac } from "../../services/rbac";

function DrawerShellForm(props: {
  title: string;
  open: boolean;
  width?: number;
  form: FormInstance;
  onClose: () => void;
  loading: boolean;
  onSubmit: () => void;
  children: React.ReactNode;
  initialValues?: Record<string, unknown>;
  /** 仅表单 + 底部「创建」，不包 Drawer（嵌入 YamlCrudPage 创建抽屉的「表单」Tab） */
  embedded?: boolean;
}) {
  const { title, open, width = 720, form, onClose, loading, onSubmit, children, initialValues, embedded } = props;
  const formNode = (
    <Form form={form} layout="vertical" requiredMark="optional" preserve={false} scrollToFirstError initialValues={initialValues}>
      {children}
    </Form>
  );
  if (embedded) {
    return (
      <>
        {formNode}
        <Space style={{ marginTop: 16 }}>
          <Button type="primary" loading={loading} onClick={() => void onSubmit()}>
            创建
          </Button>
        </Space>
      </>
    );
  }
  return (
    <Drawer
      title={title}
      placement="right"
      width={width}
      open={open}
      onClose={onClose}
      destroyOnClose
      maskClosable={false}
      styles={{ body: { paddingBottom: 24 } }}
      extra={
        <Space>
          <Button onClick={onClose}>取消</Button>
          <Button type="primary" loading={loading} onClick={() => void onSubmit()}>
            创建
          </Button>
        </Space>
      }
    >
      {formNode}
    </Drawer>
  );
}

export function NamespaceFormCreateDrawer(props: {
  open: boolean;
  onClose: () => void;
  clusterId?: number;
  onSuccess: () => void;
  embedded?: boolean;
}) {
  const { open, onClose, clusterId, onSuccess, embedded } = props;
  const [form] = Form.useForm<{ name: string; label_pairs: { key: string; value: string }[] }>();
  const [loading, setLoading] = useState(false);

  async function submit() {
    if (!clusterId) return;
    const v = await form.validateFields();
    const name = String(v.name || "").trim();
    const labels: Record<string, string> = {};
    for (const p of v.label_pairs || []) {
      const k = String(p?.key || "").trim();
      if (!k) continue;
      labels[k] = String(p?.value ?? "");
    }
    const doc: Record<string, unknown> = {
      apiVersion: "v1",
      kind: "Namespace",
      metadata: { name, ...(Object.keys(labels).length ? { labels } : {}) },
    };
    setLoading(true);
    try {
      await applyNamespace(clusterId, YAML.stringify(doc));
      message.success("命名空间已创建");
      onSuccess();
      onClose();
    } catch (e) {
      message.error(e instanceof Error ? e.message : "创建失败");
    } finally {
      setLoading(false);
    }
  }

  return (
    <DrawerShellForm
      title="表单创建 Namespace"
      open={embedded ? true : open}
      embedded={embedded}
      width={640}
      form={form}
      onClose={onClose}
      loading={loading}
      onSubmit={() => void submit()}
      initialValues={{ label_pairs: [{ key: "", value: "" }] }}
    >
      <Form.Item name="name" label="命名空间名称" rules={[{ required: true, message: "请输入名称" }]}>
        <Input placeholder="例如：team-demo" />
      </Form.Item>
      <Typography.Text type="secondary">标签（可选）</Typography.Text>
      <Form.List name="label_pairs">
        {(fields, { add, remove }) => (
          <Space direction="vertical" style={{ width: "100%", marginTop: 8 }}>
            {fields.map((f) => (
              <Space key={f.key} align="baseline">
                <Form.Item name={[f.name, "key"]} style={{ marginBottom: 0 }}>
                  <Input placeholder="键" style={{ width: 160 }} />
                </Form.Item>
                <Form.Item name={[f.name, "value"]} style={{ marginBottom: 0 }}>
                  <Input placeholder="值" style={{ width: 220 }} />
                </Form.Item>
                <Button onClick={() => remove(f.name)}>删除</Button>
              </Space>
            ))}
            <Button onClick={() => add({ key: "", value: "" })}>添加标签</Button>
          </Space>
        )}
      </Form.List>
    </DrawerShellForm>
  );
}

export function ConfigMapFormCreateDrawer(props: {
  open: boolean;
  onClose: () => void;
  clusterId?: number;
  namespace: string;
  onSuccess: () => void;
  embedded?: boolean;
}) {
  const { open, onClose, clusterId, namespace, onSuccess, embedded } = props;
  const [form] = Form.useForm<{
    name: string;
    immutable?: boolean;
    label_pairs?: { key: string; value: string }[];
    annotation_pairs?: { key: string; value: string }[];
    pairs: { key: string; value: string }[];
  }>();
  const [loading, setLoading] = useState(false);

  async function submit() {
    if (!clusterId) return;
    const v = await form.validateFields();
    const data: Record<string, string> = {};
    const labels: Record<string, string> = {};
    const annotations: Record<string, string> = {};
    for (const p of v.pairs || []) {
      const k = String(p?.key || "").trim();
      if (!k) continue;
      data[k] = String(p?.value ?? "");
    }
    for (const p of v.label_pairs || []) {
      const k = String(p?.key || "").trim();
      if (!k) continue;
      labels[k] = String(p?.value ?? "");
    }
    for (const p of v.annotation_pairs || []) {
      const k = String(p?.key || "").trim();
      if (!k) continue;
      annotations[k] = String(p?.value ?? "");
    }
    if (Object.keys(data).length === 0) {
      message.warning("请至少填写一组配置键值");
      return;
    }
    const doc = {
      apiVersion: "v1",
      kind: "ConfigMap",
      metadata: {
        name: String(v.name).trim(),
        namespace,
        ...(Object.keys(labels).length ? { labels } : {}),
        ...(Object.keys(annotations).length ? { annotations } : {}),
      },
      immutable: typeof v.immutable === "boolean" ? v.immutable : undefined,
      data,
    };
    setLoading(true);
    try {
      await applyConfigMap(clusterId, YAML.stringify(doc));
      message.success("ConfigMap 已创建");
      onSuccess();
      onClose();
    } catch (e) {
      message.error(e instanceof Error ? e.message : "创建失败");
    } finally {
      setLoading(false);
    }
  }

  return (
    <DrawerShellForm
      title="表单创建 ConfigMap"
      open={embedded ? true : open}
      embedded={embedded}
      form={form}
      onClose={onClose}
      loading={loading}
      onSubmit={() => void submit()}
      initialValues={{ pairs: [{ key: "", value: "" }], label_pairs: [{ key: "", value: "" }], annotation_pairs: [{ key: "", value: "" }] }}
    >
      <Form.Item label="目标命名空间">
        <Input value={namespace} readOnly />
      </Form.Item>
      <Form.Item name="name" label="名称" rules={[{ required: true, message: "请输入名称" }]}>
        <Input placeholder="app-config" />
      </Form.Item>
      <Form.Item name="immutable" label="Immutable" valuePropName="checked">
        <Switch />
      </Form.Item>
      <Typography.Text type="secondary">Labels（可选）</Typography.Text>
      <Form.List name="label_pairs">
        {(fields, { add, remove }) => (
          <Space direction="vertical" style={{ width: "100%", marginTop: 8 }}>
            {fields.map((f) => (
              <Space key={f.key} align="baseline">
                <Form.Item name={[f.name, "key"]} style={{ marginBottom: 0 }}><Input placeholder="label key" style={{ width: 200 }} /></Form.Item>
                <Form.Item name={[f.name, "value"]} style={{ marginBottom: 0 }}><Input placeholder="label value" style={{ width: 220 }} /></Form.Item>
                <Button onClick={() => remove(f.name)}>删除</Button>
              </Space>
            ))}
            <Button onClick={() => add({ key: "", value: "" })}>添加 Label</Button>
          </Space>
        )}
      </Form.List>
      <Typography.Text type="secondary">Annotations（可选）</Typography.Text>
      <Form.List name="annotation_pairs">
        {(fields, { add, remove }) => (
          <Space direction="vertical" style={{ width: "100%", marginTop: 8 }}>
            {fields.map((f) => (
              <Space key={f.key} align="baseline">
                <Form.Item name={[f.name, "key"]} style={{ marginBottom: 0 }}><Input placeholder="annotation key" style={{ width: 200 }} /></Form.Item>
                <Form.Item name={[f.name, "value"]} style={{ marginBottom: 0 }}><Input placeholder="annotation value" style={{ width: 220 }} /></Form.Item>
                <Button onClick={() => remove(f.name)}>删除</Button>
              </Space>
            ))}
            <Button onClick={() => add({ key: "", value: "" })}>添加 Annotation</Button>
          </Space>
        )}
      </Form.List>
      <Typography.Text type="secondary">数据键值（至少一组，值可为多行）</Typography.Text>
      <Form.List name="pairs">
        {(fields, { add, remove }) => (
          <Space direction="vertical" style={{ width: "100%", marginTop: 8 }}>
            {fields.map((f) => (
              <Space key={f.key} align="start" style={{ width: "100%" }}>
                <Form.Item name={[f.name, "key"]} rules={[{ required: true, message: "键必填" }]} style={{ marginBottom: 0 }}>
                  <Input placeholder="键" style={{ width: 200 }} />
                </Form.Item>
                <Form.Item name={[f.name, "value"]} style={{ flex: 1, marginBottom: 0, minWidth: 200 }}>
                  <Input.TextArea placeholder="值" rows={2} style={{ minHeight: 48 }} />
                </Form.Item>
                <Button onClick={() => remove(f.name)}>删除</Button>
              </Space>
            ))}
            <Button onClick={() => add({ key: "", value: "" })}>添加条目</Button>
          </Space>
        )}
      </Form.List>
    </DrawerShellForm>
  );
}

const secretTypes = [
  { label: "Opaque", value: "Opaque" },
  { label: "kubernetes.io/dockerconfigjson", value: "kubernetes.io/dockerconfigjson" },
  { label: "kubernetes.io/tls", value: "kubernetes.io/tls" },
  { label: "kubernetes.io/basic-auth", value: "kubernetes.io/basic-auth" },
  { label: "kubernetes.io/ssh-auth", value: "kubernetes.io/ssh-auth" },
];

export function SecretFormCreateDrawer(props: {
  open: boolean;
  onClose: () => void;
  clusterId?: number;
  namespace: string;
  onSuccess: () => void;
  embedded?: boolean;
}) {
  const { open, onClose, clusterId, namespace, onSuccess, embedded } = props;
  const [form] = Form.useForm<{
    name: string;
    type: string;
    data_mode?: "stringData" | "data";
    immutable?: boolean;
    label_pairs?: { key: string; value: string }[];
    annotation_pairs?: { key: string; value: string }[];
    pairs: { key: string; value: string }[];
  }>();
  const [loading, setLoading] = useState(false);

  async function submit() {
    if (!clusterId) return;
    const v = await form.validateFields();
    const stringData: Record<string, string> = {};
    const data: Record<string, string> = {};
    const labels: Record<string, string> = {};
    const annotations: Record<string, string> = {};
    for (const p of v.pairs || []) {
      const k = String(p?.key || "").trim();
      if (!k) continue;
      if ((v.data_mode || "stringData") === "data") data[k] = String(p?.value ?? "");
      else stringData[k] = String(p?.value ?? "");
    }
    for (const p of v.label_pairs || []) {
      const k = String(p?.key || "").trim();
      if (!k) continue;
      labels[k] = String(p?.value ?? "");
    }
    for (const p of v.annotation_pairs || []) {
      const k = String(p?.key || "").trim();
      if (!k) continue;
      annotations[k] = String(p?.value ?? "");
    }
    if (Object.keys(stringData).length === 0 && Object.keys(data).length === 0) {
      message.warning("请至少填写一组键值");
      return;
    }
    const doc = {
      apiVersion: "v1",
      kind: "Secret",
      metadata: {
        name: String(v.name).trim(),
        namespace,
        ...(Object.keys(labels).length ? { labels } : {}),
        ...(Object.keys(annotations).length ? { annotations } : {}),
      },
      type: v.type || "Opaque",
      immutable: typeof v.immutable === "boolean" ? v.immutable : undefined,
      ...(Object.keys(stringData).length ? { stringData } : {}),
      ...(Object.keys(data).length ? { data } : {}),
    };
    setLoading(true);
    try {
      await applySecret(clusterId, YAML.stringify(doc));
      message.success("Secret 已创建");
      onSuccess();
      onClose();
    } catch (e) {
      message.error(e instanceof Error ? e.message : "创建失败");
    } finally {
      setLoading(false);
    }
  }

  return (
    <DrawerShellForm
      title="表单创建 Secret"
      open={embedded ? true : open}
      embedded={embedded}
      form={form}
      onClose={onClose}
      loading={loading}
      onSubmit={() => void submit()}
      initialValues={{
        type: "Opaque",
        data_mode: "stringData",
        pairs: [{ key: "", value: "" }],
        label_pairs: [{ key: "", value: "" }],
        annotation_pairs: [{ key: "", value: "" }],
      }}
    >
      <Form.Item label="目标命名空间">
        <Input value={namespace} readOnly />
      </Form.Item>
      <Form.Item name="name" label="名称" rules={[{ required: true, message: "请输入名称" }]}>
        <Input placeholder="app-secret" />
      </Form.Item>
      <Form.Item name="type" label="类型" rules={[{ required: true, message: "请选择类型" }]}>
        <Select options={secretTypes} showSearch optionFilterProp="label" />
      </Form.Item>
      <Form.Item name="data_mode" label="数据模式" rules={[{ required: true }]}>
        <Select options={[{ label: "stringData（明文输入）", value: "stringData" }, { label: "data（base64）", value: "data" }]} />
      </Form.Item>
      <Form.Item name="immutable" label="Immutable" valuePropName="checked">
        <Switch />
      </Form.Item>
      <Typography.Text type="secondary">Labels（可选）</Typography.Text>
      <Form.List name="label_pairs">
        {(fields, { add, remove }) => (
          <Space direction="vertical" style={{ width: "100%", marginTop: 8 }}>
            {fields.map((f) => (
              <Space key={f.key} align="baseline">
                <Form.Item name={[f.name, "key"]} style={{ marginBottom: 0 }}><Input placeholder="label key" style={{ width: 200 }} /></Form.Item>
                <Form.Item name={[f.name, "value"]} style={{ marginBottom: 0 }}><Input placeholder="label value" style={{ width: 220 }} /></Form.Item>
                <Button onClick={() => remove(f.name)}>删除</Button>
              </Space>
            ))}
            <Button onClick={() => add({ key: "", value: "" })}>添加 Label</Button>
          </Space>
        )}
      </Form.List>
      <Typography.Text type="secondary">Annotations（可选）</Typography.Text>
      <Form.List name="annotation_pairs">
        {(fields, { add, remove }) => (
          <Space direction="vertical" style={{ width: "100%", marginTop: 8 }}>
            {fields.map((f) => (
              <Space key={f.key} align="baseline">
                <Form.Item name={[f.name, "key"]} style={{ marginBottom: 0 }}><Input placeholder="annotation key" style={{ width: 200 }} /></Form.Item>
                <Form.Item name={[f.name, "value"]} style={{ marginBottom: 0 }}><Input placeholder="annotation value" style={{ width: 220 }} /></Form.Item>
                <Button onClick={() => remove(f.name)}>删除</Button>
              </Space>
            ))}
            <Button onClick={() => add({ key: "", value: "" })}>添加 Annotation</Button>
          </Space>
        )}
      </Form.List>
      <Typography.Text type="secondary">Secret 键值（至少一组）</Typography.Text>
      <Form.List name="pairs">
        {(fields, { add, remove }) => (
          <Space direction="vertical" style={{ width: "100%", marginTop: 8 }}>
            {fields.map((f) => (
              <Space key={f.key} align="start" style={{ width: "100%" }}>
                <Form.Item name={[f.name, "key"]} rules={[{ required: true, message: "键必填" }]} style={{ marginBottom: 0 }}>
                  <Input placeholder="键" style={{ width: 200 }} />
                </Form.Item>
                <Form.Item name={[f.name, "value"]} style={{ flex: 1, marginBottom: 0 }}>
                  <Input.Password placeholder="值（敏感）" style={{ minWidth: 200 }} />
                </Form.Item>
                <Button onClick={() => remove(f.name)}>删除</Button>
              </Space>
            ))}
            <Button onClick={() => add({ key: "", value: "" })}>添加条目</Button>
          </Space>
        )}
      </Form.List>
    </DrawerShellForm>
  );
}

const pathTypes = [
  { label: "Prefix", value: "Prefix" },
  { label: "ImplementationSpecific", value: "ImplementationSpecific" },
  { label: "Exact", value: "Exact" },
];

export function IngressFormCreateDrawer(props: {
  open: boolean;
  onClose: () => void;
  clusterId?: number;
  namespace: string;
  onSuccess: () => void;
  embedded?: boolean;
}) {
  const { open, onClose, clusterId, namespace, onSuccess, embedded } = props;
  const [form] = Form.useForm<{
    name: string;
    ingress_class: string;
    annotations?: { key: string; value: string }[];
    tls?: { hosts?: string[]; secretName?: string }[];
    rules?: Array<{
      host: string;
      paths?: Array<{
        path?: string;
        path_type?: string;
        service_name?: string;
        service_port_name?: string;
        service_port_number?: number;
      }>;
    }>;
  }>();
  const [loading, setLoading] = useState(false);

  async function submit() {
    if (!clusterId) return;
    const v = await form.validateFields();
    const annotations = Object.fromEntries(
      (v.annotations ?? [])
        .map((p) => [String(p?.key ?? "").trim(), String(p?.value ?? "")] as const)
        .filter(([k]) => !!k),
    );
    const tls =
      (v.tls ?? [])
        .map((t) => ({
          hosts: (t.hosts ?? []).map((x) => String(x).trim()).filter(Boolean),
          secretName: String(t.secretName ?? "").trim() || undefined,
        }))
        .filter((t) => t.hosts.length || t.secretName) || undefined;
    const rules =
      (v.rules ?? [])
        .map((r) => ({
          host: String(r.host ?? "").trim(),
          http: {
            paths:
              (r.paths ?? [])
                .map((p) => {
                  const serviceName = String(p.service_name ?? "").trim();
                  const portName = String(p.service_port_name ?? "").trim();
                  const portNumber = typeof p.service_port_number === "number" ? p.service_port_number : undefined;
                  if (!serviceName) return null;
                  return {
                    path: String(p.path ?? "/").trim() || "/",
                    pathType: p.path_type || "Prefix",
                    backend: {
                      service: {
                        name: serviceName,
                        port: portName ? { name: portName } : { number: portNumber || 80 },
                      },
                    },
                  };
                })
                .filter(Boolean),
          },
        }))
        .filter((r) => r.host && r.http.paths.length) || undefined;
    const doc = {
      apiVersion: "networking.k8s.io/v1",
      kind: "Ingress",
      metadata: {
        name: String(v.name).trim(),
        namespace,
        ...(Object.keys(annotations).length ? { annotations } : {}),
      },
      spec: {
        ...(String(v.ingress_class || "").trim() ? { ingressClassName: String(v.ingress_class).trim() } : {}),
        rules,
        tls,
      },
    };
    setLoading(true);
    try {
      await applyIngress(clusterId, YAML.stringify(doc));
      message.success("Ingress 已创建");
      onSuccess();
      onClose();
    } catch (e) {
      message.error(e instanceof Error ? e.message : "创建失败");
    } finally {
      setLoading(false);
    }
  }

  return (
    <DrawerShellForm
      title="表单创建 Ingress"
      open={embedded ? true : open}
      embedded={embedded}
      width={760}
      form={form}
      onClose={onClose}
      loading={loading}
      onSubmit={() => void submit()}
      initialValues={{
        annotations: [{ key: "nginx.ingress.kubernetes.io/rewrite-target", value: "/" }],
        tls: [{ hosts: [], secretName: "" }],
        rules: [{ host: "", paths: [{ path: "/", path_type: "Prefix", service_name: "", service_port_number: 80 }] }],
      }}
    >
      <Form.Item label="目标命名空间">
        <Input value={namespace} readOnly />
      </Form.Item>
      <Form.Item name="name" label="名称" rules={[{ required: true, message: "请输入名称" }]}>
        <Input placeholder="demo-ingress" />
      </Form.Item>
      <Form.Item name="ingress_class" label="IngressClass 名称" extra="与集群 Ingress Controller 一致，如 nginx；可选">
        <Input placeholder="nginx" />
      </Form.Item>
      <Typography.Text type="secondary">Ingress 注解（可选）</Typography.Text>
      <Form.List name="annotations">
        {(fields, { add, remove }) => (
          <Space direction="vertical" style={{ width: "100%", marginTop: 8 }}>
            {fields.map((f) => (
              <Space key={f.key} align="baseline">
                <Form.Item name={[f.name, "key"]} style={{ marginBottom: 0 }}>
                  <Input placeholder="annotation key" style={{ width: 280 }} />
                </Form.Item>
                <Form.Item name={[f.name, "value"]} style={{ marginBottom: 0 }}>
                  <Input placeholder="annotation value" style={{ width: 320 }} />
                </Form.Item>
                <Button onClick={() => remove(f.name)}>删除</Button>
              </Space>
            ))}
            <Button onClick={() => add({ key: "", value: "" })}>添加注解</Button>
          </Space>
        )}
      </Form.List>
      <Typography.Text type="secondary" style={{ marginTop: 12, display: "block" }}>
        TLS（可选）
      </Typography.Text>
      <Form.List name="tls">
        {(fields, { add, remove }) => (
          <Space direction="vertical" style={{ width: "100%", marginTop: 8 }}>
            {fields.map((f) => (
              <Space key={f.key} align="baseline" style={{ width: "100%" }}>
                <Form.Item name={[f.name, "hosts"]} label="Hosts" style={{ flex: 1 }}>
                  <Select mode="tags" tokenSeparators={[",", " "]} placeholder="example.com,api.example.com" />
                </Form.Item>
                <Form.Item name={[f.name, "secretName"]} label="SecretName" style={{ flex: 1 }}>
                  <Input placeholder="tls-secret" />
                </Form.Item>
                <Button onClick={() => remove(f.name)}>删除</Button>
              </Space>
            ))}
            <Button onClick={() => add({ hosts: [], secretName: "" })}>添加 TLS</Button>
          </Space>
        )}
      </Form.List>
      <Typography.Text type="secondary" style={{ marginTop: 12, display: "block" }}>
        Rules
      </Typography.Text>
      <Form.List name="rules">
        {(ruleFields, { add: addRule, remove: removeRule }) => (
          <Space direction="vertical" style={{ width: "100%", marginTop: 8 }}>
            {ruleFields.map((rf) => (
              <div key={rf.key} style={{ border: "1px solid #f0f0f0", padding: 12, borderRadius: 8 }}>
                <Space style={{ width: "100%" }} align="start">
                  <Form.Item name={[rf.name, "host"]} label="域名 (Host)" rules={[{ required: true, message: "请输入域名" }]} style={{ flex: 1 }}>
                    <Input placeholder="app.example.com" />
                  </Form.Item>
                  <Button onClick={() => removeRule(rf.name)}>删除 Rule</Button>
                </Space>
                <Form.List name={[rf.name, "paths"]}>
                  {(pathFields, { add: addPath, remove: removePath }) => (
                    <Space direction="vertical" style={{ width: "100%" }}>
                      {pathFields.map((pf) => (
                        <div key={pf.key} style={{ border: "1px dashed #d9d9d9", padding: 10, borderRadius: 6 }}>
                          <Space style={{ width: "100%" }} align="start">
                            <Form.Item name={[pf.name, "path"]} label="路径" style={{ flex: 1 }}>
                              <Input placeholder="/" />
                            </Form.Item>
                            <Form.Item name={[pf.name, "path_type"]} label="路径匹配" style={{ width: 220 }}>
                              <Select options={pathTypes} />
                            </Form.Item>
                            <Button onClick={() => removePath(pf.name)}>删除 Path</Button>
                          </Space>
                          <Space style={{ width: "100%" }} align="start">
                            <Form.Item name={[pf.name, "service_name"]} label="后端 Service 名称" rules={[{ required: true, message: "请输入 Service" }]} style={{ flex: 1 }}>
                              <Input placeholder="my-service" />
                            </Form.Item>
                            <Form.Item name={[pf.name, "service_port_name"]} label="Service 端口名" style={{ width: 220 }}>
                              <Input placeholder="http" />
                            </Form.Item>
                            <Form.Item name={[pf.name, "service_port_number"]} label="Service 端口号" style={{ width: 220 }}>
                              <InputNumber min={1} max={65535} style={{ width: "100%" }} />
                            </Form.Item>
                          </Space>
                        </div>
                      ))}
                      <Button onClick={() => addPath({ path: "/", path_type: "Prefix", service_name: "", service_port_number: 80 })}>添加 Path</Button>
                    </Space>
                  )}
                </Form.List>
              </div>
            ))}
            <Button onClick={() => addRule({ host: "", paths: [{ path: "/", path_type: "Prefix", service_name: "", service_port_number: 80 }] })}>添加 Rule</Button>
          </Space>
        )}
      </Form.List>
    </DrawerShellForm>
  );
}

const apiGroupOptions = [
  { label: "核心组 (\"\")", value: "" },
  { label: "apps", value: "apps" },
  { label: "batch", value: "batch" },
  { label: "networking.k8s.io", value: "networking.k8s.io" },
  { label: "rbac.authorization.k8s.io", value: "rbac.authorization.k8s.io" },
];

const resourceOptions = ["pods", "services", "configmaps", "secrets", "deployments", "statefulsets", "daemonsets", "ingresses", "nodes", "namespaces", "events", "endpoints", "persistentvolumeclaims", "persistentvolumes"].map((r) => ({
  label: r,
  value: r,
}));

const verbOptions = ["get", "list", "watch", "create", "update", "patch", "delete"].map((v) => ({ label: v, value: v }));

export function RbacRoleFormCreateDrawer(props: {
  open: boolean;
  onClose: () => void;
  clusterId?: number;
  namespace: string;
  onSuccess: () => void;
  embedded?: boolean;
}) {
  const { open, onClose, clusterId, namespace, onSuccess, embedded } = props;
  const [form] = Form.useForm<{ name: string; api_group: string; resources: string[]; verbs: string[] }>();
  const [loading, setLoading] = useState(false);

  async function submit() {
    if (!clusterId) return;
    const v = await form.validateFields();
    const rules = [
      {
        apiGroups: [v.api_group === "" ? "" : v.api_group],
        resources: v.resources?.length ? v.resources : ["pods"],
        verbs: v.verbs?.length ? v.verbs : ["get", "list"],
      },
    ];
    const doc = {
      apiVersion: "rbac.authorization.k8s.io/v1",
      kind: "Role",
      metadata: { name: String(v.name).trim(), namespace },
      rules,
    };
    setLoading(true);
    try {
      await applyRbac(clusterId, YAML.stringify(doc));
      message.success("Role 已创建");
      onSuccess();
      onClose();
    } catch (e) {
      message.error(e instanceof Error ? e.message : "创建失败");
    } finally {
      setLoading(false);
    }
  }

  return (
    <DrawerShellForm
      title="表单创建 Role"
      open={embedded ? true : open}
      embedded={embedded}
      form={form}
      onClose={onClose}
      loading={loading}
      onSubmit={() => void submit()}
      initialValues={{ api_group: "", resources: ["pods"], verbs: ["get", "list"] }}
    >
      <Form.Item label="命名空间">
        <Input value={namespace} readOnly />
      </Form.Item>
      <Form.Item name="name" label="Role 名称" rules={[{ required: true, message: "请输入名称" }]}>
        <Input placeholder="demo-role" />
      </Form.Item>
      <Form.Item name="api_group" label="API 组" rules={[{ required: true }]}>
        <Select options={apiGroupOptions} />
      </Form.Item>
      <Form.Item name="resources" label="资源" rules={[{ required: true, message: "请选择资源" }]}>
        <Select mode="multiple" options={resourceOptions} placeholder="选择资源类型" optionFilterProp="label" />
      </Form.Item>
      <Form.Item name="verbs" label="动词" rules={[{ required: true, message: "请选择动词" }]}>
        <Select mode="multiple" options={verbOptions} />
      </Form.Item>
    </DrawerShellForm>
  );
}

export function RbacClusterRoleFormCreateDrawer(props: {
  open: boolean;
  onClose: () => void;
  clusterId?: number;
  onSuccess: () => void;
  embedded?: boolean;
}) {
  const { open, onClose, clusterId, onSuccess, embedded } = props;
  const [form] = Form.useForm<{ name: string; api_group: string; resources: string[]; verbs: string[] }>();
  const [loading, setLoading] = useState(false);

  async function submit() {
    if (!clusterId) return;
    const v = await form.validateFields();
    const rules = [
      {
        apiGroups: [v.api_group === "" ? "" : v.api_group],
        resources: v.resources?.length ? v.resources : ["nodes"],
        verbs: v.verbs?.length ? v.verbs : ["get", "list"],
      },
    ];
    const doc = {
      apiVersion: "rbac.authorization.k8s.io/v1",
      kind: "ClusterRole",
      metadata: { name: String(v.name).trim() },
      rules,
    };
    setLoading(true);
    try {
      await applyRbac(clusterId, YAML.stringify(doc));
      message.success("ClusterRole 已创建");
      onSuccess();
      onClose();
    } catch (e) {
      message.error(e instanceof Error ? e.message : "创建失败");
    } finally {
      setLoading(false);
    }
  }

  return (
    <DrawerShellForm
      title="表单创建 ClusterRole"
      open={embedded ? true : open}
      embedded={embedded}
      form={form}
      onClose={onClose}
      loading={loading}
      onSubmit={() => void submit()}
      initialValues={{ api_group: "", resources: ["nodes"], verbs: ["get", "list"] }}
    >
      <Form.Item name="name" label="ClusterRole 名称" rules={[{ required: true, message: "请输入名称" }]}>
        <Input placeholder="demo-clusterrole" />
      </Form.Item>
      <Form.Item name="api_group" label="API 组" rules={[{ required: true }]}>
        <Select options={apiGroupOptions} />
      </Form.Item>
      <Form.Item name="resources" label="资源" rules={[{ required: true }]}>
        <Select mode="multiple" options={resourceOptions} optionFilterProp="label" />
      </Form.Item>
      <Form.Item name="verbs" label="动词" rules={[{ required: true }]}>
        <Select mode="multiple" options={verbOptions} />
      </Form.Item>
    </DrawerShellForm>
  );
}

const subjectKindOptions = [
  { label: "User", value: "User" },
  { label: "ServiceAccount", value: "ServiceAccount" },
  { label: "Group", value: "Group" },
];

export function RbacRoleBindingFormCreateDrawer(props: {
  open: boolean;
  onClose: () => void;
  clusterId?: number;
  namespace: string;
  onSuccess: () => void;
  embedded?: boolean;
}) {
  const { open, onClose, clusterId, namespace, onSuccess, embedded } = props;
  const [form] = Form.useForm<{
    name: string;
    role_kind: "Role" | "ClusterRole";
    role_name: string;
    subject_kind: string;
    subject_name: string;
    subject_namespace: string;
  }>();
  const [loading, setLoading] = useState(false);

  async function submit() {
    if (!clusterId) return;
    const v = await form.validateFields();
    const sub: Record<string, unknown> = { kind: v.subject_kind, name: String(v.subject_name).trim() };
    if (v.subject_kind === "ServiceAccount" && String(v.subject_namespace || "").trim()) {
      sub.namespace = String(v.subject_namespace).trim();
    }
    const doc = {
      apiVersion: "rbac.authorization.k8s.io/v1",
      kind: "RoleBinding",
      metadata: { name: String(v.name).trim(), namespace },
      subjects: [sub],
      roleRef: {
        apiGroup: "rbac.authorization.k8s.io",
        kind: v.role_kind,
        name: String(v.role_name).trim(),
      },
    };
    setLoading(true);
    try {
      await applyRbac(clusterId, YAML.stringify(doc));
      message.success("RoleBinding 已创建");
      onSuccess();
      onClose();
    } catch (e) {
      message.error(e instanceof Error ? e.message : "创建失败");
    } finally {
      setLoading(false);
    }
  }

  return (
    <DrawerShellForm
      title="表单创建 RoleBinding"
      open={embedded ? true : open}
      embedded={embedded}
      width={760}
      form={form}
      onClose={onClose}
      loading={loading}
      onSubmit={() => void submit()}
      initialValues={{ role_kind: "Role", subject_kind: "User" }}
    >
      <Form.Item label="命名空间">
        <Input value={namespace} readOnly />
      </Form.Item>
      <Form.Item name="name" label="RoleBinding 名称" rules={[{ required: true, message: "请输入名称" }]}>
        <Input placeholder="demo-binding" />
      </Form.Item>
      <Space style={{ width: "100%" }} align="start">
        <Form.Item name="role_kind" label="引用类型" rules={[{ required: true }]} style={{ width: 200 }}>
          <Select options={[{ label: "Role", value: "Role" }, { label: "ClusterRole", value: "ClusterRole" }]} />
        </Form.Item>
        <Form.Item name="role_name" label="角色名称" rules={[{ required: true, message: "请输入角色名" }]} style={{ flex: 1 }}>
          <Input placeholder="demo-role" />
        </Form.Item>
      </Space>
      <Form.Item name="subject_kind" label="主体类型" rules={[{ required: true }]}>
        <Select options={subjectKindOptions} />
      </Form.Item>
      <Form.Item name="subject_name" label="主体名称" rules={[{ required: true, message: "请输入用户名/SA 名" }]}>
        <Input placeholder="alice 或 default/my-sa" />
      </Form.Item>
      <Form.Item name="subject_namespace" label="ServiceAccount 所在命名空间" extra="仅当主体为 ServiceAccount 时填写">
        <Input placeholder="default" />
      </Form.Item>
    </DrawerShellForm>
  );
}

export function RbacClusterRoleBindingFormCreateDrawer(props: {
  open: boolean;
  onClose: () => void;
  clusterId?: number;
  onSuccess: () => void;
  embedded?: boolean;
}) {
  const { open, onClose, clusterId, onSuccess, embedded } = props;
  const [form] = Form.useForm<{
    name: string;
    role_name: string;
    subject_kind: string;
    subject_name: string;
    subject_namespace: string;
  }>();
  const [loading, setLoading] = useState(false);

  async function submit() {
    if (!clusterId) return;
    const v = await form.validateFields();
    const sub: Record<string, unknown> = { kind: v.subject_kind, name: String(v.subject_name).trim() };
    if (v.subject_kind === "ServiceAccount" && String(v.subject_namespace || "").trim()) {
      sub.namespace = String(v.subject_namespace).trim();
    }
    const doc = {
      apiVersion: "rbac.authorization.k8s.io/v1",
      kind: "ClusterRoleBinding",
      metadata: { name: String(v.name).trim() },
      subjects: [sub],
      roleRef: {
        apiGroup: "rbac.authorization.k8s.io",
        kind: "ClusterRole",
        name: String(v.role_name).trim(),
      },
    };
    setLoading(true);
    try {
      await applyRbac(clusterId, YAML.stringify(doc));
      message.success("ClusterRoleBinding 已创建");
      onSuccess();
      onClose();
    } catch (e) {
      message.error(e instanceof Error ? e.message : "创建失败");
    } finally {
      setLoading(false);
    }
  }

  return (
    <DrawerShellForm
      title="表单创建 ClusterRoleBinding"
      open={embedded ? true : open}
      embedded={embedded}
      width={760}
      form={form}
      onClose={onClose}
      loading={loading}
      onSubmit={() => void submit()}
      initialValues={{ subject_kind: "User" }}
    >
      <Form.Item name="name" label="名称" rules={[{ required: true, message: "请输入名称" }]}>
        <Input placeholder="demo-crb" />
      </Form.Item>
      <Form.Item name="role_name" label="ClusterRole 名称" rules={[{ required: true, message: "请输入 ClusterRole 名" }]}>
        <Input />
      </Form.Item>
      <Form.Item name="subject_kind" label="主体类型" rules={[{ required: true }]}>
        <Select options={subjectKindOptions} />
      </Form.Item>
      <Form.Item name="subject_name" label="主体名称" rules={[{ required: true, message: "请输入" }]}>
        <Input />
      </Form.Item>
      <Form.Item name="subject_namespace" label="ServiceAccount 命名空间" extra="可选">
        <Input />
      </Form.Item>
    </DrawerShellForm>
  );
}
