import * as IconComponents from "@ant-design/icons";
import type { MenuProps } from "antd";
import { Link } from "react-router-dom";
import type { ReactNode } from "react";
import type { MenuItem } from "../services/menus";
import { LoginOutlined, HistoryOutlined, ApiOutlined } from "@ant-design/icons";

export type AntdMenuItem = NonNullable<MenuProps["items"]>[number];

const LOG_MENU_ITEMS: AntdMenuItem[] = [
  { key: "/login-logs", icon: <LoginOutlined />, label: <Link to="/login-logs">登录日志</Link> },
  { key: "/operation-logs", icon: <HistoryOutlined />, label: <Link to="/operation-logs">操作历史</Link> },
];

const LOG_MENU_KEYS = new Set(["/login-logs", "/operation-logs"]);

const BANNED_MENU_ITEM: AntdMenuItem = {
  key: "/banned-ips",
  icon: <ApiOutlined />,
  label: <Link to="/banned-ips">封禁 IP 管理</Link>,
};

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
  return ensureLogMenus(items);
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

function ensureLogMenus(items: AntdMenuItem[]): AntdMenuItem[] {
  // Try to find the logical "系统管理" group: the first top-level item whose children
  // already contain login/operation logs. If found, ensure the log items and the banned-ip
  // item exist under that group. Otherwise, fallback to previous behavior of injecting
  // log items into existing parent nodes that have children.
  const itemsCopy = items.map((i) => i) as AntdMenuItem[];

  let systemIndex = -1;
  // Prefer explicit `/system` top-level group if present
  for (let i = 0; i < itemsCopy.length; i++) {
    const candidate = itemsCopy[i];
    if (candidate && typeof candidate === "object" && "key" in candidate && String(candidate.key) === "/system") {
      systemIndex = i;
      break;
    }
  }

  if (systemIndex === -1) {
    for (let i = 0; i < itemsCopy.length; i++) {
    const it = itemsCopy[i];
    if (!it || typeof it !== "object" || !("children" in it) || !Array.isArray(it.children)) continue;
    const childKeys = new Set((it.children as AntdMenuItem[]).map((c) => (c && typeof c === "object" && "key" in c ? String(c.key) : "")));
    for (const lk of LOG_MENU_KEYS) {
      if (childKeys.has(lk)) {
        systemIndex = i;
        break;
      }
    }
    if (systemIndex !== -1) break;
  }
  }
  if (systemIndex !== -1) {
    const it = itemsCopy[systemIndex] as any;
    const children: AntdMenuItem[] = Array.isArray(it.children) ? [...(it.children as AntdMenuItem[])] : [];
    const childKeys = new Set(children.map((c: any) => String(c?.key ?? "")));
    for (const logItem of LOG_MENU_ITEMS) {
      const logKey = String((logItem as any).key ?? "");
      if (!childKeys.has(logKey)) {
        children.push(logItem);
        childKeys.add(logKey);
      }
    }
    const bannedKey = String((BANNED_MENU_ITEM as any).key ?? "");
    if (!childKeys.has(bannedKey)) {
      children.push(BANNED_MENU_ITEM);
    }
    itemsCopy[systemIndex] = { ...(it as AntdMenuItem), children } as AntdMenuItem;
    return itemsCopy;
  }

  // fallback: inject log items into any parent that has children (previous behavior)
  return items.map((item) => {
    if (!item || typeof item !== "object" || !("children" in item)) return item;
    const children = Array.isArray(item.children) ? [...(item.children as AntdMenuItem[])] : [];
    const childKeys = new Set(children.map((c) => (c && typeof c === "object" && "key" in c ? String(c.key) : "")));
    for (const logItem of LOG_MENU_ITEMS) {
      const logKey = logItem && typeof logItem.key === "string" ? logItem.key : String(logItem?.key);
      if (!childKeys.has(logKey)) {
        children.push(logItem);
      }
    }
    return { ...item, children } as AntdMenuItem;
  });
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
