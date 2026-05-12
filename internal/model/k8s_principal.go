package model

// K8s 集群授权主体类型（对齐 k8m：用户、用户组；yunshu 同时保留角色模板批量授权）。
const (
	K8sPrincipalRole = "role"
	K8sPrincipalUser = "user"
	K8sPrincipalGroup = "group"
)
