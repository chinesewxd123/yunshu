import * as IconComponents from "@ant-design/icons";
import type { MenuProps } from "antd";
import { Link } from "react-router-dom";
import type { ReactNode } from "react";
import type { MenuItem } from "../services/menus";

export type AntdMenuItem = NonNullable<MenuProps["items"]>[number];

function menuIconByName(name?: string): ReactNode {
  if (!name?.trim()) return undefined;
  const Cmp = (IconComponents as unknown as Record<string, React.ComponentType<object>>)[name.trim()];
  if (!Cmp) return undefined;
  return <Cmp />;
}

function filterVisible(menus: MenuItem[]): MenuItem[] {
  return menus
    .filter((m) => m.status === 1 && !m.hidden)
    .sort((a, b) => (a.sort ?? 0) - (b.sort ?? 0))
    .map((m) => ({
      ...m,
      children: m.children?.length ? filterVisible(m.children) : undefined,
    }));
}

/** 将后端菜单树转为 Ant Design Menu items（与 React Router 联动） */
export function buildSiderMenuItems(menus: MenuItem[]): AntdMenuItem[] {
  const nodes = filterVisible(menus);
  const items = nodes.map((m) => toAntdItem(m));
  // 菜单以数据库为准，不在前端侧做“兜底注入”。
  return dedupeMenuByKey(items);
}

function toAntdItem(m: MenuItem): AntdMenuItem {
  const icon = menuIconByName(m.icon);
  const children = m.children?.length ? buildSiderMenuItems(m.children) : undefined;
  if (children?.length) {
    return {
      key: m.path?.trim() ? m.path : `menu-${m.id}`,
      icon,
      label: m.name,
      children,
    };
  }
  const to = m.path?.trim() ?? "/";
  return {
    key: to,
    icon,
    label: <Link to={to}>{m.name}</Link>,
  };
}

function normalizeMenuKey(key: string): string {
  const raw = key.trim().toLowerCase();
  const cleaned = raw.replace(/\/+$/, "");
  if (cleaned === "/pod" || cleaned === "/pods") return "/pods";
  if (cleaned === "/cluster" || cleaned === "/clusters") return "/clusters";
  return cleaned || "/";
}

function dedupeMenuByKey(items: AntdMenuItem[], globalSeen?: Set<string>): AntdMenuItem[] {
  const seen = globalSeen ?? new Set<string>();
  const out: AntdMenuItem[] = [];
  for (const item of items) {
    if (!item || typeof item !== "object" || !("key" in item)) continue;
    const key = normalizeMenuKey(String(item.key));
    if (seen.has(key)) continue;
    seen.add(key);
    if ("children" in item && Array.isArray(item.children)) {
      out.push({ ...item, children: dedupeMenuByKey(item.children as AntdMenuItem[], seen) } as AntdMenuItem);
      continue;
    }
    out.push(item);
  }
  return out;
}

// collectAllKeys removed — no longer needed after refactor

export function flattenMenuTitles(menus: MenuItem[], acc: Record<string, string> = {}): Record<string, string> {
  for (const m of menus) {
    if (m.path?.trim()) {
      acc[m.path] = m.name;
    }
    if (m.children?.length) {
      flattenMenuTitles(m.children, acc);
    }
  }
  return acc;
}

function collectKeys(items: AntdMenuItem[]): string[] {
  const out: string[] = [];
  for (const it of items) {
    if (!it || typeof it !== "object" || !("key" in it)) continue;
    out.push(String(it.key));
    if ("children" in it && Array.isArray(it.children)) {
      out.push(...collectKeys(it.children as AntdMenuItem[]));
    }
  }
  return out;
}

/** 根据当前 path 匹配最具体的菜单 key（用于 selectedKeys） */
export function matchMenuSelectedKey(pathname: string, items: AntdMenuItem[]): string {
  const keys = collectKeys(items).filter((k) => k && !k.startsWith("menu-"));
  if (pathname === "/") {
    return keys.includes("/") ? "/" : pathname;
  }
  let best = "/";
  for (const k of keys) {
    if (k === "/") continue;
    if (pathname === k || pathname.startsWith(`${k}/`)) {
      if (k.length > best.length) best = k;
    }
  }
  return best;
}
