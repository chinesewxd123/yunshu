package k8sauth

import (
	"strconv"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
)

// PrincipalPack 集群权限匹配主体（角色 + 用户 + 用户组），对齐 k8m。
type PrincipalPack struct {
	RoleCodes  []string
	UserID     uint
	GroupCodes []string
}

// PackFromCurrentUser 从登录上下文构造。
func PackFromCurrentUser(u *auth.CurrentUser) PrincipalPack {
	if u == nil {
		return PrincipalPack{}
	}
	roles := dedupeStrings(u.RoleCodes)
	groups := dedupeStrings(u.GroupCodes)
	return PrincipalPack{
		RoleCodes:  roles,
		UserID:     u.ID,
		GroupCodes: groups,
	}
}

func dedupeStrings(in []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, c := range in {
		s := strings.TrimSpace(c)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

// UserRefString 用户主体 principal_ref。
func UserRefString(userID uint) string {
	return strconv.FormatUint(uint64(userID), 10)
}

// PrincipalRows 展开为 (kind, ref) 用于 SQL。
func (p PrincipalPack) PrincipalRows() []struct{ Kind, Ref string } {
	var out []struct{ Kind, Ref string }
	for _, rc := range p.RoleCodes {
		out = append(out, struct{ Kind, Ref string }{model.K8sPrincipalRole, rc})
	}
	if p.UserID > 0 {
		out = append(out, struct{ Kind, Ref string }{model.K8sPrincipalUser, UserRefString(p.UserID)})
	}
	for _, gc := range p.GroupCodes {
		out = append(out, struct{ Kind, Ref string }{model.K8sPrincipalGroup, gc})
	}
	return out
}
