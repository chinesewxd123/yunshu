import type { FormInstance } from "antd";
import { useState } from "react";

export type WorkloadFormCtx = { clusterId: number; namespace: string; name?: string };
export type WorkloadFormMode = "create" | "edit";

export type WorkloadDetailLike = { yaml?: string; object?: any };

export type UseWorkloadFormActionsOptions<FV> = {
  form: FormInstance<FV>;
  mode?: boolean; // whether expose create/edit mode
  defaultMode?: WorkloadFormMode;
  getDetail: (clusterId: number, namespace: string, name: string) => Promise<WorkloadDetailLike>;
  toFormValues: (detail: WorkloadDetailLike) => FV | null | undefined;
  buildFallbackValues: (args: { recordName: string; namespace: string; record?: any }) => Partial<FV>;
};

export function useWorkloadFormActions<FV>(opts: UseWorkloadFormActionsOptions<FV>) {
  const [open, setOpen] = useState(false);
  const [loading, setLoading] = useState(false);
  const [ctx, setCtx] = useState<WorkloadFormCtx | null>(null);
  const [mode, setMode] = useState<WorkloadFormMode>(opts.defaultMode ?? "create");

  const close = () => setOpen(false);

  /** 仅填充表单与 ctx，不打开独立 Drawer（用于 YamlCrudPage 内嵌「表单创建」Tab） */
  const prepareCreate = (next: { clusterId: number; namespace: string }, initial?: Partial<FV>) => {
    if (opts.mode) setMode("create");
    setCtx({ clusterId: next.clusterId, namespace: next.namespace });
    opts.form.resetFields();
    if (initial && Object.keys(initial).length > 0) {
      opts.form.setFieldsValue(initial as any);
    }
    setLoading(false);
  };

  const openCreate = (next: { clusterId: number; namespace: string }, initial?: Partial<FV>) => {
    prepareCreate(next, initial);
    setOpen(true);
  };

  const openEdit = (next: { clusterId: number; namespace: string; name: string }, record?: any) => {
    if (opts.mode) setMode("edit");
    setCtx({ clusterId: next.clusterId, namespace: next.namespace, name: next.name });
    setOpen(true);
    setLoading(true);
    void (async () => {
      try {
        const d = await opts.getDetail(next.clusterId, next.namespace, next.name);
        const fv = opts.toFormValues(d);
        if (fv) {
          // namespace follows toolbar namespace when present
          opts.form.setFieldsValue({ ...(fv as any), namespace: next.namespace } as any);
        } else {
          opts.form.setFieldsValue(opts.buildFallbackValues({ recordName: next.name, namespace: next.namespace, record }) as any);
        }
      } finally {
        setLoading(false);
      }
    })();
  };

  return { open, loading, ctx, mode, setMode, prepareCreate, openCreate, openEdit, close, setOpen, setLoading, setCtx };
}

