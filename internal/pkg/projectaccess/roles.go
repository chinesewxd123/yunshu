package projectaccess

import "strings"

// RoleRank 项目内角色权重：数值越大权限越高。
func RoleRank(role string) int {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "owner":
		return 4
	case "admin":
		return 3
	case "member":
		return 2
	case "readonly":
		return 1
	default:
		return 2
	}
}

// RoleAtLeast 若 subject 角色权重不低于 need，返回 true。
func RoleAtLeast(subjectRole, need string) bool {
	return RoleRank(subjectRole) >= RoleRank(need)
}

// IsReadonly 是否为项目只读成员。
func IsReadonly(role string) bool {
	return strings.EqualFold(strings.TrimSpace(role), "readonly")
}
