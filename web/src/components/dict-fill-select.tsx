import { Form, Select, message } from "antd";
import type { FormInstance } from "antd/es/form";
import { useEffect, useMemo, useState } from "react";
import { revealDictEntryValue } from "../services/dict";

export type DictFillOption = { label: string; value: string | number; id?: number; sensitive?: boolean };

type DictFillSelectProps = {
  form: FormInstance;
  fieldName: string;
  options: DictFillOption[];
  placeholder?: string;
  allowClear?: boolean;
  disabled?: boolean;
  style?: React.CSSProperties;
};

function norm(v: unknown): string {
  return String(v ?? "").trim();
}

/**
 * 使用规范：
 * - DictFillSelect：只有一个业务字段，直接存字典 value（例如 url、token、agentId）。
 * - DictLabelFillSelect：需要“按 label 选、按 value 存”，并且希望持久化 label 用于稳定回显。
 */
export function DictFillSelect({ form, fieldName, options, placeholder, allowClear = true, disabled, style }: DictFillSelectProps) {
  const raw = Form.useWatch(fieldName, form);
  const [pickedSensitiveId, setPickedSensitiveId] = useState<number | undefined>();

  useEffect(() => {
    setPickedSensitiveId(undefined);
  }, [fieldName]);

  const selectValue = useMemo(() => {
    if (pickedSensitiveId != null) return pickedSensitiveId;
    const s = norm(raw);
    if (!s) return undefined;
    const hit = options.find((o) => !o.sensitive && norm(o.value) === s);
    return hit ? hit.value : undefined;
  }, [raw, options, pickedSensitiveId]);

  return (
    <Select
      allowClear={allowClear}
      disabled={disabled}
      placeholder={placeholder}
      style={style}
      options={options.map((o) => ({
        label: o.label,
        value: o.sensitive && o.id != null ? o.id : o.value,
      }))}
      value={selectValue}
      onChange={async (v) => {
        if (v === undefined || v === null || v === "") {
          setPickedSensitiveId(undefined);
          form.setFieldValue(fieldName, "");
          return;
        }
        const byId = options.find((o) => o.sensitive && o.id === v);
        if (byId?.id != null) {
          const hide = message.loading("正在获取字典值…", 0);
          try {
            const { value } = await revealDictEntryValue(byId.id);
            setPickedSensitiveId(byId.id);
            form.setFieldValue(fieldName, value);
          } catch (e) {
            message.error(e instanceof Error ? e.message : String(e));
          } finally {
            hide();
          }
          return;
        }
        setPickedSensitiveId(undefined);
        form.setFieldValue(fieldName, v === undefined || v === null ? "" : v);
      }}
    />
  );
}

type DictLabelFillSelectProps = {
  form: FormInstance;
  /** 存储“字典标签”的字段（用于稳定回显） */
  labelFieldName: string;
  /** 业务真实字段（存 value） */
  targetFieldName: string;
  options: DictFillOption[];
  placeholder?: string;
  allowClear?: boolean;
  disabled?: boolean;
  style?: React.CSSProperties;
};

/**
 * 适用于“按 label 选、按 value 存”的场景：
 * - 选择时：同时写 labelField + targetField
 * - 回显时：优先 labelField，其次按 targetField(value) 反推 label
 * - 手改 targetField 且与已选 label 不一致时：自动清空 labelField，避免假回显
 */
export function DictLabelFillSelect({
  form,
  labelFieldName,
  targetFieldName,
  options,
  placeholder,
  allowClear = true,
  disabled,
  style,
}: DictLabelFillSelectProps) {
  const selectedLabelRaw = Form.useWatch(labelFieldName, form);
  const targetRaw = Form.useWatch(targetFieldName, form);
  const selectedLabel = norm(selectedLabelRaw);
  const targetValueNorm = norm(targetRaw);

  const selectValue = useMemo(() => {
    if (selectedLabel) {
      const byLabel = options.find((o) => norm(o.label) === selectedLabel);
      if (byLabel) return byLabel.label;
    }
    if (!targetValueNorm) return undefined;
    const byValue = options.find((o) => !o.sensitive && norm(o.value) === targetValueNorm);
    return byValue ? byValue.label : undefined;
  }, [selectedLabel, targetValueNorm, options]);

  useEffect(() => {
    if (!selectedLabel) return;
    const byLabel = options.find((o) => norm(o.label) === selectedLabel);
    if (!byLabel) {
      form.setFieldValue(labelFieldName, undefined);
      return;
    }
    if (byLabel.sensitive) {
      return;
    }
    if (targetValueNorm && norm(byLabel.value) !== targetValueNorm) {
      form.setFieldValue(labelFieldName, undefined);
    }
  }, [selectedLabel, targetValueNorm, options, form, labelFieldName]);

  return (
    <Select
      allowClear={allowClear}
      disabled={disabled}
      placeholder={placeholder}
      style={style}
      options={options.map((o) => ({ label: o.label, value: o.label }))}
      value={selectValue}
      onChange={async (label) => {
        if (!label) {
          form.setFieldsValue({ [labelFieldName]: undefined, [targetFieldName]: "" });
          return;
        }
        const hit = options.find((o) => norm(o.label) === norm(label));
        if (!hit) return;
        if (hit.sensitive && hit.id != null) {
          const hide = message.loading("正在获取字典值…", 0);
          try {
            const { value } = await revealDictEntryValue(hit.id);
            form.setFieldsValue({
              [labelFieldName]: hit.label,
              [targetFieldName]: value,
            });
          } catch (e) {
            message.error(e instanceof Error ? e.message : String(e));
          } finally {
            hide();
          }
          return;
        }
        form.setFieldsValue({
          [labelFieldName]: hit.label,
          [targetFieldName]: hit.value,
        });
      }}
    />
  );
}
