import { Alert, Card, Result, Spin, Typography } from "antd";
import { Suspense, useEffect, useMemo, useState } from "react";
import { Link, useLocation } from "react-router-dom";
import { getMenuTree } from "../services/menus";
import type { MenuItem } from "../services/menus";
import { createLazyMenuPage } from "../utils/menu-page-loader";
import { findMenuByPath, normalizeMenuPath } from "../utils/menu-path";

function RouteFallback() {
  return (
    <div className="page-loading">
      <Spin size="large" />
    </div>
  );
}

export function DynamicMenuPage() {
  const location = useLocation();
  const [menus, setMenus] = useState<MenuItem[] | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    void (async () => {
      try {
        const tree = await getMenuTree();
        if (!cancelled) setMenus(tree ?? []);
      } catch (e) {
        if (!cancelled) setLoadError(e instanceof Error ? e.message : "加载菜单失败");
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  const menuItem = useMemo(() => {
    if (!menus?.length) return undefined;
    return findMenuByPath(menus, location.pathname);
  }, [menus, location.pathname]);

  const LazyComp = useMemo(() => {
    const c = menuItem?.component?.trim();
    if (!c) return null;
    return createLazyMenuPage(c);
  }, [menuItem?.component]);

  if (loadError) {
    return <Result status="error" title="菜单加载失败" subTitle={loadError} />;
  }

  if (menus === null) {
    return <RouteFallback />;
  }

  if (!menuItem) {
    return (
      <Result
        status="404"
        title="未找到菜单"
        subTitle={`当前地址 ${location.pathname} 未在「菜单管理」中配置，或已被隐藏/停用。`}
        extra={
          <Link to="/menus">
            <Typography.Link>前往菜单管理</Typography.Link>
          </Link>
        }
      />
    );
  }

  if (menuItem.children && menuItem.children.length > 0) {
    return (
      <Card className="table-card" title={menuItem.name}>
        <Typography.Paragraph type="secondary">这是目录菜单，请从左侧选择具体子菜单进入页面。</Typography.Paragraph>
        <ul style={{ margin: 0, paddingLeft: 20 }}>
          {menuItem.children
            .filter((c) => c.status === 1 && !c.hidden)
            .map((c) => (
              <li key={c.id}>
                <Link to={normalizeMenuPath(c.path)}>{c.name}</Link>
                <Typography.Text type="secondary"> {c.path}</Typography.Text>
              </li>
            ))}
        </ul>
      </Card>
    );
  }

  if (!menuItem.component?.trim()) {
    return (
      <Card className="table-card">
        <Result
          status="info"
          title="未配置前端组件"
          subTitle={
            <span>
              请在「菜单管理」中为该菜单填写 <Typography.Text code>组件路径</Typography.Text>（例如 <Typography.Text code>containerd-page</Typography.Text>
              ），并新增对应文件 <Typography.Text code>src/pages/containerd-page.tsx</Typography.Text>，导出{" "}
              <Typography.Text code>ContainerdPage</Typography.Text>。
            </span>
          }
        />
      </Card>
    );
  }

  if (!LazyComp) {
    return (
      <Card className="table-card">
        <Result
          status="warning"
          title="未找到页面文件"
          subTitle={
            <span>
              已填写 component 为「{menuItem.component}」，但未找到匹配的 <Typography.Text code>src/pages/**/*-page.tsx</Typography.Text>。
              请新建文件并导出与文件名对应的 PascalCase 组件名。
            </span>
          }
          extra={
            <Alert
              type="info"
              showIcon
              message="命名约定"
              description="例如 component 填 foo-bar-page，则需存在 src/pages/foo-bar-page.tsx，且 export function FooBarPage() { ... }"
            />
          }
        />
      </Card>
    );
  }

  return (
    <Suspense fallback={<RouteFallback />}>
      <LazyComp />
    </Suspense>
  );
}
