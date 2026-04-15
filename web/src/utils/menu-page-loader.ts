import { lazy, type ComponentType, type LazyExoticComponent } from "react";

/**
 * 构建期扫描 src/pages 下所有 *-page.tsx，运行时按菜单 component 字段懒加载。
 * 约定：文件名 xxx-page.tsx 需导出 PascalCase，如 foo-page → FooPage。
 */
const pageLoaders = import.meta.glob("../pages/**/*-page.tsx") as Record<string, () => Promise<Record<string, ComponentType<object>>>>;

/** 供菜单表单下拉：当前工程内所有 *-page 组件 id（与 component 字段一致，如 dashboard-page、foo/bar-page） */
export function listMenuPageComponentIds(): string[] {
  return Object.keys(pageLoaders)
    .map((key) => key.replace(/^\.\.\/pages\//, "").replace(/\.tsx$/i, ""))
    .filter(Boolean)
    .sort((a, b) => a.localeCompare(b));
}

function toExportNameFromFilename(fileBase: string): string {
  const base = fileBase.replace(/\.tsx$/i, "");
  return base
    .split("-")
    .map((s) => s.charAt(0).toUpperCase() + s.slice(1))
    .join("");
}

function resolveGlobKey(componentField: string): string | undefined {
  const normalized = componentField
    .trim()
    .replace(/^\//, "")
    .replace(/\\/g, "/")
    .replace(/^src\/pages\//, "")
    .replace(/^pages\//, "")
    .replace(/\.tsx$/i, "");

  if (!/^[\w/-]+$/.test(normalized)) {
    return undefined;
  }

  const expectedId = normalized.endsWith("-page") ? normalized : `${normalized}-page`;
  const key = Object.keys(pageLoaders).find((k) => k.replace(/^\.\.\/pages\//, "").replace(/\.tsx$/i, "") === expectedId);
  return key;
}

/**
 * @param componentField 菜单中的 component，如 `containerd-page` 或 `foo/bar-page`（对应 `src/pages/foo/bar-page.tsx`）
 */
export function createLazyMenuPage(componentField: string): LazyExoticComponent<ComponentType<object>> | null {
  const key = resolveGlobKey(componentField);
  if (!key || !pageLoaders[key]) {
    return null;
  }

  const fileBase = key.split("/").pop()!.replace(/\.tsx$/i, "");
  const exportName = toExportNameFromFilename(fileBase);

  return lazy(async () => {
    const mod = await pageLoaders[key]();
    const named = mod as Record<string, ComponentType<object>>;
    const direct = named[exportName];
    const caseInsensitive = Object.entries(named).find(([k, v]) => {
      if (!v || k === "default") return false;
      return k.toLowerCase() === exportName.toLowerCase();
    })?.[1];
    const firstPageLike = Object.entries(named).find(([k, v]) => {
      if (!v || k === "default") return false;
      return /page$/i.test(k);
    })?.[1];
    const Comp = direct ?? caseInsensitive ?? (mod as { default?: ComponentType<object> }).default ?? firstPageLike;
    if (!Comp) {
      throw new Error(`页面模块未导出「${exportName}」或 default：${key}`);
    }
    return { default: Comp };
  });
}
