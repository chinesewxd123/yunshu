import * as IconComponents from "@ant-design/icons";
import { createElement, type ComponentType, type ReactNode } from "react";

const EXCLUDE = new Set([
  "default",
  "createFromIconfontCN",
  "IconProvider",
  "getTwoToneColor",
  "setTwoToneColor",
]);

export function listAntdIconNames(): string[] {
  return Object.keys(IconComponents)
    .filter(
      (k) =>
        !EXCLUDE.has(k) &&
        /^[A-Z]/.test(k) &&
        (k.endsWith("Outlined") || k.endsWith("Filled") || k.endsWith("TwoTone")),
    )
    .sort((a, b) => a.localeCompare(b));
}

let cachedOptions: { value: string; label: ReactNode }[] | null = null;

/** 带图标预览的 Select 选项（模块级缓存，避免重复创建 VNode） */
export function getAntdIconSelectOptions(): { value: string; label: ReactNode }[] {
  if (cachedOptions) return cachedOptions;
  const names = listAntdIconNames();
  const icons = IconComponents as unknown as Record<string, ComponentType<object>>;
  cachedOptions = names.map((name) => {
    const Cmp = icons[name];
    return {
      value: name,
      label: Cmp
        ? createElement(
            "span",
            { style: { display: "inline-flex", alignItems: "center", gap: 8 } },
            createElement(Cmp),
            createElement("span", null, name),
          )
        : name,
    };
  });
  return cachedOptions;
}
