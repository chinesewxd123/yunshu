package service

import (
	"fmt"

	"yunshu/internal/model"

	"github.com/casbin/casbin/v2"
)

func UserSubject(userID uint) string {
	return fmt.Sprintf("user:%d", userID)
}

func SyncUserRoles(enforcer *casbin.SyncedEnforcer, userID uint, roles []model.Role) error {
	subject := UserSubject(userID)
	if _, err := enforcer.DeleteRolesForUser(subject); err != nil {
		return err
	}
	for _, role := range roles {
		if _, err := enforcer.AddRoleForUser(subject, role.Code); err != nil {
			return err
		}
	}
	return nil
}

func ReplaceRoleCode(enforcer *casbin.SyncedEnforcer, oldCode, newCode string) error {
	if oldCode == newCode {
		return nil
	}

	policies := enforcer.GetFilteredPolicy(0, oldCode)
	for _, policy := range policies {
		if len(policy) < 3 {
			continue
		}
		if _, err := enforcer.RemovePolicy(policy[0], policy[1], policy[2]); err != nil {
			return err
		}
		if _, err := enforcer.AddPolicy(newCode, policy[1], policy[2]); err != nil {
			return err
		}
	}

	groupings := enforcer.GetFilteredGroupingPolicy(1, oldCode)
	for _, grouping := range groupings {
		if len(grouping) < 2 {
			continue
		}
		if _, err := enforcer.RemoveGroupingPolicy(grouping[0], oldCode); err != nil {
			return err
		}
		if _, err := enforcer.AddGroupingPolicy(grouping[0], newCode); err != nil {
			return err
		}
	}
	return nil
}

func RemoveRolePolicies(enforcer *casbin.SyncedEnforcer, roleCode string) error {
	if _, err := enforcer.DeletePermissionsForUser(roleCode); err != nil {
		return err
	}
	if _, err := enforcer.DeleteRolesForUser(roleCode); err != nil {
		return err
	}
	return nil
}

func ReplacePermissionResource(enforcer *casbin.SyncedEnforcer, oldResource, oldAction, newResource, newAction string) error {
	if oldResource == newResource && oldAction == newAction {
		return nil
	}

	policies := enforcer.GetFilteredPolicy(1, oldResource, oldAction)
	for _, policy := range policies {
		if len(policy) < 3 {
			continue
		}
		if _, err := enforcer.RemovePolicy(policy[0], oldResource, oldAction); err != nil {
			return err
		}
		if _, err := enforcer.AddPolicy(policy[0], newResource, newAction); err != nil {
			return err
		}
	}
	return nil
}

func RemovePermissionPolicies(enforcer *casbin.SyncedEnforcer, resource, action string) error {
	_, err := enforcer.RemoveFilteredPolicy(1, resource, action)
	return err
}
