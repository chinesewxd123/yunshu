import type { MenuItem } from "../services/menus";

export function normalizeMenuPath(path: string): string {
  if (!path?.trim()) return "/";
  const p = path.startsWith("/") ? path : `/${path}`;
  return p.replace(/\/$/, "") || "/";
}

export function flattenMenuItems(menus: MenuItem[]): MenuItem[] {
  const out: MenuItem[] = [];
  function walk(list: MenuItem[]) {
    for (const m of list) {
      out.push(m);
      if (m.children?.length) walk(m.children);
    }
  }
  walk(menus);
  return out;
}

/** 按当前 URL 在菜单树中查找节点（含目录节点） */
export function findMenuByPath(menus: MenuItem[], pathname: string): MenuItem | undefined {
  const p = normalizeMenuPath(pathname);
  const flat = flattenMenuItems(menus);
  return flat.find((m) => normalizeMenuPath(m.path) === p);
}
