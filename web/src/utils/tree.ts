import type { TreeProps } from "antd";
import type { PermissionItem, RoleItem } from "../types/api";

export type AppTreeData = NonNullable<TreeProps["treeData"]>;

export function buildRoleTreeData(roles: RoleItem[]): AppTreeData {
  const enabledRoles = roles.filter((role) => role.status === 1);
  const disabledRoles = roles.filter((role) => role.status !== 1);
  const treeData: AppTreeData = [];

  if (enabledRoles.length > 0) {
    treeData.push({
      key: "roles-enabled",
      title: `启用角色 (${enabledRoles.length})`,
      selectable: false,
      disableCheckbox: true,
      children: enabledRoles.map((role) => ({
        key: role.id,
        value: role.id,
        title: `${role.name} (${role.code})`,
      })),
    });
  }

  if (disabledRoles.length > 0) {
    treeData.push({
      key: "roles-disabled",
      title: `停用角色 (${disabledRoles.length})`,
      selectable: false,
      disableCheckbox: true,
      children: disabledRoles.map((role) => ({
        key: role.id,
        value: role.id,
        title: `${role.name} (${role.code})`,
      })),
    });
  }

  return treeData;
}

export function buildPermissionTreeData(permissions: PermissionItem[]): AppTreeData {
  const moduleMap = new Map<string, Map<string, PermissionItem[]>>();

  for (const permission of permissions) {
    const moduleName = getModuleName(permission.resource);
    if (!moduleMap.has(moduleName)) {
      moduleMap.set(moduleName, new Map<string, PermissionItem[]>());
    }
    const resourceMap = moduleMap.get(moduleName)!;
    if (!resourceMap.has(permission.resource)) {
      resourceMap.set(permission.resource, []);
    }
    resourceMap.get(permission.resource)!.push(permission);
  }

  return Array.from(moduleMap.entries())
    .sort(([left], [right]) => left.localeCompare(right))
    .map(([moduleName, resourceMap]) => ({
      key: `module:${moduleName}`,
      title: moduleName,
      selectable: false,
      disableCheckbox: true,
      children: Array.from(resourceMap.entries())
        .sort(([left], [right]) => left.localeCompare(right))
        .map(([resource, items]) => ({
          key: `resource:${resource}`,
          title: resource,
          selectable: false,
          disableCheckbox: true,
          children: items
            .slice()
            .sort((left, right) => {
              const actionCompare = left.action.localeCompare(right.action);
              if (actionCompare !== 0) {
                return actionCompare;
              }
              return left.name.localeCompare(right.name);
            })
            .map((permission) => ({
              key: permission.id,
              value: permission.id,
              title: `${permission.action} · ${permission.name}`,
            })),
        })),
    }));
}

export function normalizeCheckedKeys(checkedKeys: Parameters<NonNullable<TreeProps["onCheck"]>>[0]): number[] {
  const rawKeys = Array.isArray(checkedKeys) ? checkedKeys : checkedKeys.checked;
  return rawKeys
    .map((item) => Number(item))
    .filter((item) => Number.isInteger(item));
}

function getModuleName(resource: string) {
  const segments = resource.split("/").filter(Boolean);
  return segments[2] ?? segments[segments.length - 1] ?? resource;
}
