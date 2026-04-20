import { Form, Select } from "antd";
import type { FormInstance } from "antd/es/form";
import { useEffect, useMemo } from "react";

export type DictFillOption = { label: string; value: string | number };

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
  const selectValue = useMemo(() => {
    const s = norm(raw);
    if (!s) return undefined;
    const hit = options.find((o) => norm(o.value) === s);
    return hit ? hit.value : undefined;
  }, [raw, options]);

  return (
    <Select
      allowClear={allowClear}
      disabled={disabled}
      placeholder={placeholder}
      style={style}
      options={options.map((o) => ({ label: o.label, value: o.value }))}
      value={selectValue}
      onChange={(v) => {
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
    const byValue = options.find((o) => norm(o.value) === targetValueNorm);
    return byValue ? byValue.label : undefined;
  }, [selectedLabel, targetValueNorm, options]);

  useEffect(() => {
    if (!selectedLabel) return;
    const byLabel = options.find((o) => norm(o.label) === selectedLabel);
    if (!byLabel) {
      form.setFieldValue(labelFieldName, undefined);
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
      onChange={(label) => {
        if (!label) {
          form.setFieldsValue({ [labelFieldName]: undefined });
          return;
        }
        const hit = options.find((o) => norm(o.label) === norm(label));
        if (!hit) return;
        form.setFieldsValue({
          [labelFieldName]: hit.label,
          [targetFieldName]: hit.value,
        });
      }}
    />
  );
}
