// Package constants：统一业务错误码（数字 error_code）、产品话术、以及脚本生成的 ErrMsg* 文案与模板。
// 业务码分段：10xxx 通用；11xxx 请求校验；20xxx–26xxx 按功能域手写；11020/10901 等为可变文案固定码（Err*WithMsg，可传 ErrMsg* / fmt 拼接）。
// 请使用 BizError、域内 Err*；长尾或脚本 ErrMsg* 用 Err*WithMsg(constants.ErrMsg…)。response.Error(c, err) 将业务码写入 JSON error_code。
// 文件后部「固定错误/提示文案」「fmt.Sprintf 模板」为脚本生成区，勿手改常量值。
package constants

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"yunshu/internal/pkg/apperror"
)

// BizError 构造业务错误：HTTP 状态、数字业务码、产品话术（error_code 为数字字符串）。
func BizError(httpStatus, bizCode int, message string) error {
	return apperror.New(httpStatus, message, strconv.Itoa(bizCode))
}

// ErrBadRequestWithMsg 固定业务码 11020，文案由调用方传入（绑定失败、fmt 拼接等）。
func ErrBadRequestWithMsg(msg string) error {
	return BizError(http.StatusBadRequest, 11020, msg)
}

// ErrNotFoundWithMsg 固定业务码 11021。
func ErrNotFoundWithMsg(msg string) error {
	return BizError(http.StatusNotFound, 11021, msg)
}

// ErrForbiddenWithMsg 固定业务码 11022。
func ErrForbiddenWithMsg(msg string) error {
	return BizError(http.StatusForbidden, 11022, msg)
}

// ErrUnauthorizedWithMsg 固定业务码 11023。
func ErrUnauthorizedWithMsg(msg string) error {
	return BizError(http.StatusUnauthorized, 11023, msg)
}

// ErrConflictWithMsg 固定业务码 11024。
func ErrConflictWithMsg(msg string) error {
	return BizError(http.StatusConflict, 11024, msg)
}

// ErrInternalWithMsg 固定业务码 10901（与 ErrInternal 固定话术区分）。
func ErrInternalWithMsg(msg string) error {
	return BizError(http.StatusInternalServerError, 10901, msg)
}

// ErrTooManyRequestsWithMsg 固定业务码 10902。
func ErrTooManyRequestsWithMsg(msg string) error {
	return BizError(http.StatusTooManyRequests, 10902, msg)
}

// —— 通用 10xxx ——
var (
	ErrBadRequest               = BizError(http.StatusBadRequest, 10001, "请求参数无效，请检查后重试")
	ErrUnauthorized             = BizError(http.StatusUnauthorized, 10002, "登录已失效或凭证无效，请重新登录")
	ErrForbidden                = BizError(http.StatusForbidden, 10003, "当前账号无权执行该操作")
	ErrNotFound                 = BizError(http.StatusNotFound, 10004, "所请求的资源不存在或已删除")
	ErrConflict                 = BizError(http.StatusConflict, 10005, "资源状态冲突，请刷新后重试")
	ErrInternal                 = BizError(http.StatusInternalServerError, 10006, "平台服务异常，请稍后重试或联系管理员")
	ErrTooManyRequests          = BizError(http.StatusTooManyRequests, 10007, "操作过于频繁，请稍后再试")
	ErrMissingAuthHeader        = BizError(http.StatusUnauthorized, 10008, "缺少或非法的鉴权请求头，请先登录或检查客户端配置")
	ErrAccessTokenInvalid       = BizError(http.StatusUnauthorized, 10009, "访问令牌无效，请重新登录")
	ErrLoginSessionExpired      = BizError(http.StatusUnauthorized, 10010, "登录会话已过期，请重新登录")
	ErrAccountPrincipalNotFound = BizError(http.StatusUnauthorized, 10011, "账号不存在或已删除，请检查登录信息")
	ErrAccountDisabled          = BizError(http.StatusForbidden, 10012, "账号已被禁用，如需协助请联系管理员")
	ErrWSMissingTokenParam      = BizError(http.StatusUnauthorized, 10013, "缺少连接令牌参数，请在请求中携带 token")
	ErrNotLoggedIn              = BizError(http.StatusUnauthorized, 10014, "未完成登录，请先登录后再访问该资源")
)

// —— 请求校验 11xxx ——
var (
	ErrIncludeRegexInvalid = BizError(http.StatusBadRequest, 11001, "「包含」筛选条件格式不正确，请检查正则表达式")
	ErrExcludeRegexInvalid = BizError(http.StatusBadRequest, 11002, "「排除」筛选条件格式不正确，请检查正则表达式")
	ErrInvalidRequestParam = BizError(http.StatusBadRequest, 11003, "请求参数不合法，请检查后重试")
)

// —— 认证与账号 20xxx ——
var (
	ErrUserNotFound            = BizError(http.StatusNotFound, 20001, "用户信息不存在或已停用")
	ErrPasswordIncorrect       = BizError(http.StatusUnauthorized, 20002, "账号或密码不正确，请检查后重试")
	ErrEmailAlreadyRegistered  = BizError(http.StatusConflict, 20003, "该邮箱已被占用，请更换后重试")
	ErrTokenGenerate           = BizError(http.StatusInternalServerError, 20004, "登录凭证签发失败，请稍后重试")
	ErrEmailNotBound           = BizError(http.StatusBadRequest, 20005, "当前账号未绑定邮箱，请先在安全设置中完成绑定")
	ErrNicknameRequired        = BizError(http.StatusBadRequest, 20006, "昵称不能为空，请填写后保存")
	ErrCaptchaIPRateLimited    = BizError(http.StatusTooManyRequests, 20007, "验证码请求过于频繁，请稍后再试")
	ErrCaptchaExpired          = BizError(http.StatusBadRequest, 20008, "验证码已失效，请重新获取")
	ErrCaptchaIncorrect        = BizError(http.StatusBadRequest, 20009, "验证码不正确，请检查后重试")
	ErrCaptchaRequired         = BizError(http.StatusBadRequest, 20010, "请先填写验证码")
	ErrCaptchaInvalidOrExpired = BizError(http.StatusUnauthorized, 20011, "验证码无效或已过期，请重新获取")
	ErrUsernameTaken           = BizError(http.StatusConflict, 20012, "用户名已被占用，请更换后重试")
	ErrCaptchaCoolingDown      = BizError(http.StatusConflict, 20013, "验证码已发送，请稍后再试")
)

// —— 日志采集 Agent 21xxx ——
var (
	ErrAgentTokenInvalid          = BizError(http.StatusUnauthorized, 21001, "Agent 访问凭证无效或已轮换，请重新下发 Token")
	ErrAgentRegisterClosed        = BizError(http.StatusForbidden, 21002, "公共 Agent 自助注册已关闭，请联系管理员")
	ErrServerDisabledForAgent     = BizError(http.StatusForbidden, 21003, "目标服务器已停用，无法完成 Agent 相关操作")
	ErrAgentRegisterSecretInvalid = BizError(http.StatusUnauthorized, 21004, "Agent 注册密钥无效，请核对平台侧配置")
	ErrAgentTokenMissing          = BizError(http.StatusUnauthorized, 21005, "缺少 Agent 访问令牌，请携带 Token 后重试")
)

// —— 告警 22xxx ——
var (
	ErrAlertSilenceNotFound     = BizError(http.StatusNotFound, 22001, "告警静默规则不存在或已失效")
	ErrAlertWebhookTokenInvalid = BizError(http.StatusForbidden, 22002, "告警回调令牌无效，请核对告警集成配置")
)

// —— 项目与日志源 23xxx ——
var (
	ErrProjectNotFound             = BizError(http.StatusNotFound, 23001, "项目不存在或已被删除")
	ErrLogSourceServerNotFound     = BizError(http.StatusNotFound, 23002, "日志源服务器不存在或已移除")
	ErrServerNotInCurrentProject   = BizError(http.StatusBadRequest, 23003, "该服务器不属于当前项目，请切换项目后再操作")
	ErrServerProjectMismatch       = BizError(http.StatusBadRequest, 23004, "项目与日志源归属不一致，请刷新后重试")
	ErrProjectIDRequired           = BizError(http.StatusBadRequest, 23005, "缺少项目标识（project_id），请补充后重试")
	ErrNameRequired                = BizError(http.StatusBadRequest, 23006, "名称不能为空，请填写后提交")
	ErrUploadFailed                = BizError(http.StatusBadRequest, 23007, "文件上传失败，请检查网络或文件大小后重试")
	ErrServerNotInProject          = BizError(http.StatusBadRequest, 23008, "该服务器不在当前项目中，请刷新或切换范围后重试")
	ErrServerNotInProjectForbidden = BizError(http.StatusForbidden, 23009, "该服务器不在当前项目中，当前账号无权访问")
	ErrProjectMemberRequired         = BizError(http.StatusForbidden, 23010, "您不是该项目的成员，无权访问或操作该项目资源")
	ErrProjectAdminRequired          = BizError(http.StatusForbidden, 23011, "该操作需要项目管理员或负责人权限")
	ErrProjectReadonlyMember         = BizError(http.StatusForbidden, 23012, "项目只读成员不能执行此修改类操作")
	ErrK8sClusterProjectAccessDenied  = BizError(http.StatusForbidden, 23013, "该集群已绑定到其他业务项目，当前账号不在允许范围内")
)

// —— RBAC / 组织 24xxx ——
var (
	ErrRoleNotFound       = BizError(http.StatusNotFound, 24001, "角色不存在或已删除")
	ErrUserGroupNotFound  = BizError(http.StatusNotFound, 24002, "用户组不存在或已删除")
	ErrPermissionNotFound = BizError(http.StatusNotFound, 24005, "权限项不存在或已变更")
	ErrMenuNotFound       = BizError(http.StatusNotFound, 24003, "菜单不存在或已下线")
	ErrDepartmentNotFound = BizError(http.StatusNotFound, 24004, "部门不存在或已撤销")
)

// —— 自助注册 25xxx ——
var (
	ErrRegistrationRequestNotFound  = BizError(http.StatusNotFound, 25001, "注册申请不存在或已处理")
	ErrRegistrationAlreadyProcessed = BizError(http.StatusConflict, 25002, "该注册申请已审核，请勿重复操作")
	ErrRegistrationDuplicatePending = BizError(http.StatusConflict, 25003, "该用户名或邮箱已有待审核申请，请勿重复提交")
)

// —— Kubernetes 集群 26xxx ——
var (
	// ErrK8sNamespaceAlreadyExists 表单创建命名空间时名称已在集群中存在（HTTP 409 / error_code 26001）。
	ErrK8sNamespaceAlreadyExists = BizError(http.StatusConflict, 26001, "该命名空间已存在，请勿重复创建")
)

// ErrK8sNamespaceAlreadyExistsMsg 返回业务码 26001，文案包含冲突的名称。
func ErrK8sNamespaceAlreadyExistsMsg(name string) error {
	n := strings.TrimSpace(name)
	if n == "" {
		return ErrK8sNamespaceAlreadyExists
	}
	return BizError(http.StatusConflict, 26001, fmt.Sprintf("命名空间「%s」已存在，请勿重复创建", n))
}

// 展示用语（非 error）：Agent 离线归因话术
const (
	LogAgentOfflineNeverConnected = "从未连接成功"
	LogAgentOfflineHeartbeatLost  = "心跳超时（失联或进程僵死）"
	LogAgentOfflineAgentStopped   = "Agent 已停止"
	LogAgentOfflineAgentError     = "Agent 异常（进程上报 error）"
)

// 文案模板与拼接前缀
const (
	ErrFmtJSONFieldMustBeObject   = "%s 须为 JSON 对象字符串，请检查配置或请求体"
	ErrFmtAlertSilenceBatchEndsAt = "批量静默项「%s」结束时间须晚于开始时间"
	ErrMsgSSHConnectFailedPrefix  = "远程连接失败（SSH）："
	ErrMsgSSHExecFailedPrefix     = "远程命令执行失败："
	ErrMsgCloudSDKPrefix          = "云平台返回异常："
)

// —— 固定错误/提示文案 ——
const (
	// AK/SK 未配置
	ErrMsg2a88bfc17d34 = "AK/SK 未配置"
	// Agent 令牌无效
	ErrMsgc3157b56d185 = "Agent 令牌无效"
	// CR 资源不存在
	ErrMsg0646b6d79607 = "CR 资源不存在"
	// CRD 资源不存在
	ErrMsgfc34c3a3e621 = "CRD 资源不存在"
	// ConfigMap 资源不存在
	ErrMsgd42231f0a030 = "ConfigMap 资源不存在"
	// CronJob 资源不存在
	ErrMsgc6ae960d40d1 = "CronJob 资源不存在"
	// DaemonSet 资源不存在
	ErrMsg728030d27854 = "DaemonSet 资源不存在"
	// Deployment 资源不存在
	ErrMsgf6d026c4bc20 = "Deployment 资源不存在"
	// Excel 文件不合法
	ErrMsg04d13e805997 = "Excel 文件不合法"
	// Ingress 资源不存在
	ErrMsg82a55c47e927 = "Ingress 资源不存在"
	// IngressClass 资源不存在
	ErrMsgeb6e8490034b = "IngressClass 资源不存在"
	// Job 资源不存在
	ErrMsg656deb688b72 = "Job 资源不存在"
	// K8s 客户端不存在
	ErrMsgc674e8a0802b = "K8s 客户端不存在"
	// K8s 集群实例不存在
	ErrMsg5248c9e19a3f = "K8s 集群实例不存在"
	// K8s 集群重连失败
	ErrMsgb9cf6d1a2c2e = "K8s 集群重连失败"
	// NetworkPolicy 资源不存在
	ErrMsge64b05879667 = "NetworkPolicy 资源不存在"
	// Pod 不存在，无法编辑
	ErrMsg86c3cb5f9474 = "Pod 不存在，无法编辑"
	// Pod 名称不合法：必须为 RFC1123 subdomain（小写字母/数字/短横线/点，且首尾为字母或数字）
	ErrMsgb75b5021a60c = "Pod 名称不合法：必须为 RFC1123 subdomain（小写字母/数字/短横线/点，且首尾为字母或数字）"
	// Pod 正在删除中（Terminating），请稍后再试
	ErrMsgcc094caf1644 = "Pod 正在删除中（Terminating），请稍后再试"
	// Secret 资源不存在
	ErrMsg2859a961fa4c = "Secret 资源不存在"
	// Service 不存在
	ErrMsg951c363660f3 = "Service 不存在"
	// ServiceAccount 不存在
	ErrMsg510ffa989afc = "ServiceAccount 不存在"
	// StatefulSet 资源不存在
	ErrMsg728d3e3b08a7 = "StatefulSet 资源不存在"
	// Token 无效
	ErrMsg6b5348b62040 = "Token 无效"
	// action 不能为空
	ErrMsg62812cadd1e4 = "action 不能为空"
	// actions 与 paths 不能为空
	ErrMsg316eb6f964a1 = "actions 与 paths 不能为空"
	// cannot change project_id
	ErrMsgd165b6af9d52 = "cannot change project_id"
	// cloud_instance_id 为空：请先通过「同步云账号」导入云服务器，或在服务器编辑里补充实例ID
	ErrMsg7d2f738475f2 = "cloud_instance_id 为空：请先通过「同步云账号」导入云服务器，或在服务器编辑里补充实例ID"
	// cloud_region 为空：请先同步云账号或在服务器编辑里补充地域
	ErrMsgd5a14bd7dce0 = "cloud_region 为空：请先同步云账号或在服务器编辑里补充地域"
	// ends_at 必须晚于 starts_at
	ErrMsgc1f741f96c03 = "ends_at 必须晚于 starts_at"
	// equal_labels_json 必须是字符串数组JSON
	ErrMsg814a89cb6d8c = "equal_labels_json 必须是字符串数组JSON"
	// group/version/resource 不能为空
	ErrMsgf757c3be22a2 = "group/version/resource 不能为空"
	// headers_json 解析失败，请检查 JSON 格式
	ErrMsg2a14724b02d9 = "headers_json 解析失败，请检查 JSON 格式"
	// kind/name 不能为空
	ErrMsg462a4f71f455 = "kind/name 不能为空"
	// match_regex_json 格式错误
	ErrMsgdd9901e7c511 = "match_regex_json 格式错误"
	// matcher name 不能为空
	ErrMsg3f8f3c7674a1 = "matcher name 不能为空"
	// name required
	ErrMsg65788b1abed6 = "name required"
	// namespace/name 不能为空
	ErrMsge278df185255 = "namespace/name 不能为空"
	// new_password 不能为空
	ErrMsgaef828b00100 = "new_password 不能为空"
	// project_id 不能为空
	ErrMsga23e0337e347 = "project_id 不能为空"
	// project_id 必填
	ErrMsg9a7f154a70af = "project_id 必填"
	// receiver group not found
	ErrMsg7628a50dd0ab = "receiver group not found"
	// replicas 不能小于 0
	ErrMsgba0d4ada9f12 = "replicas 不能小于 0"
	// status 只能为 0 或 1
	ErrMsg394db01d16f3 = "status 只能为 0 或 1"
	// 上级部门不存在
	ErrMsg1e390c7912c1 = "上级部门不存在"
	// 上级部门不能选择自己
	ErrMsg0915d23f388b = "上级部门不能选择自己"
	// 不支持的 RBAC 类型
	ErrMsgf5e00f884bae = "不支持的 RBAC 类型"
	// 不支持的 action
	ErrMsg1707715d174a = "不支持的 action"
	// 不支持的 kind
	ErrMsga4f3daa0fb94 = "不支持的 kind"
	// 不支持的云厂商
	ErrMsg4e7f045ccd87 = "不支持的云厂商"
	// 不支持的工作负载类型
	ErrMsgd5692b195622 = "不支持的工作负载类型"
	// 不能修改订阅节点所属项目
	ErrMsg586f4a40fa8c = "不能修改订阅节点所属项目"
	// 不能将节点移动到自己的子树下
	ErrMsg431b9ea19dd4 = "不能将节点移动到自己的子树下"
	// 不能将节点设为自身的父节点
	ErrMsg657954176c7d = "不能将节点设为自身的父节点"
	// 不能将部门移动到其子级部门下
	ErrMsg29de0c2b961b = "不能将部门移动到其子级部门下"
	// 云到期规则不存在
	ErrMsg34cc3b1e5427 = "云到期规则不存在"
	// 云服务器缺少 group_id
	ErrMsg5f86cac1154b = "云服务器缺少 group_id"
	// 云账号 AK/SK 未配置
	ErrMsgedfdf2d93904 = "云账号 AK/SK 未配置"
	// 云账号不存在
	ErrMsgd19fc495559f = "云账号不存在"
	// 云账号不属于当前项目
	ErrMsg053a6a395b16 = "云账号不属于当前项目"
	// 仅 prometheus 数据源支持查询
	ErrMsg9a8a590cfc72 = "仅 prometheus 数据源支持查询"
	// 仅支持云服务器操作
	ErrMsg1f9244d53fae = "仅支持云服务器操作"
	// 令牌不能为空
	ErrMsg488a8dce8ef5 = "令牌不能为空"
	// 企业微信应用模式至少需要配置 atMobiles/atUserIds/isAtAll
	ErrMsg3963f2e4d87c = "企业微信应用模式至少需要配置 atMobiles/atUserIds/isAtAll"
	// 企业微信应用模式需配置 corpID/corpSecret/agentId
	ErrMsg5fcdf3f22c91 = "企业微信应用模式需配置 corpID/corpSecret/agentId"
	// 公共 Agent 注册已关闭
	ErrMsg4b401fb484fe = "公共 Agent 注册已关闭"
	// 凭据认证类型不合法
	ErrMsge9e731f82ff9 = "凭据认证类型不合法"
	// 分组不属于当前项目
	ErrMsg757ed9cbc3d5 = "分组不属于当前项目"
	// 包含正则表达式不合法
	ErrMsg1e7f0cdb6585 = "包含正则表达式不合法"
	// 参数不合法
	ErrMsgfc77bc7bce4d = "参数不合法"
	// 同字典类型下该值已存在，请勿重复保存
	ErrMsg1ffcbfd43034 = "同字典类型下该值已存在，请勿重复保存"
	// 同字典类型下该值已存在，请勿重复创建
	ErrMsg9ea86777037d = "同字典类型下该值已存在，请勿重复创建"
	// 同字典类型下该标签已存在，请勿重复保存
	ErrMsg47f29b52ac8f = "同字典类型下该标签已存在，请勿重复保存"
	// 同字典类型下该标签已存在，请勿重复创建
	ErrMsg7e043b9a81af = "同字典类型下该标签已存在，请勿重复创建"
	// 同级部门名称已存在
	ErrMsg6719d7537f54 = "同级部门名称已存在"
	// 名称不能为空
	ErrMsge83654164a44 = "名称不能为空"
	// 告警 webhook 入队失败：序列化错误
	ErrMsg39d72e4b8516 = "告警 webhook 入队失败：序列化错误"
	// 告警 webhook 队列已满，请稍后重试
	ErrMsgfd7c760c8d45 = "告警 webhook 队列已满，请稍后重试"
	// 告警回调令牌无效
	ErrMsg10f21831d3c0 = "告警回调令牌无效"
	// 告警数据源不存在
	ErrMsg2f3e2fbecdc5 = "告警数据源不存在"
	// 命令不能为空
	ErrMsgeddb4f63e4c7 = "命令不能为空"
	// 命名空间不存在
	ErrMsg52d9e6e7f573 = "命名空间不存在"
	// 命名空间不能为空
	ErrMsgc67b07d6cf4d = "命名空间不能为空"
	// 命名空间和名称不能为空
	ErrMsgd7d6f9e01fb2 = "命名空间和名称不能为空"
	// 处理人配置不存在
	ErrMsg8faff6dbdd1d = "处理人配置不存在"
	// 字典条目不存在
	ErrMsg094b285159a4 = "字典条目不存在"
	// 字典类型、标签和值不能为空
	ErrMsg0adab830348f = "字典类型、标签和值不能为空"
	// 容器名称不合法：必须为 RFC1123 label（小写字母/数字/短横线，且首尾为字母或数字）
	ErrMsg819b22fb354c = "容器名称不合法：必须为 RFC1123 label（小写字母/数字/短横线，且首尾为字母或数字）"
	// 密码不能为空
	ErrMsg3aa7a9e035a4 = "密码不能为空"
	// 已存在启用且未过期的同类静默规则，无需重复创建；如需调整请编辑现有静默
	ErrMsg612f94e277ef = "已存在启用且未过期的同类静默规则，无需重复创建；如需调整请编辑现有静默"
	// 已存在启用且未过期的同类静默规则，请勿重复创建（批量任务已中止）
	ErrMsg76b99177ec58 = "已存在启用且未过期的同类静默规则，请勿重复创建（批量任务已中止）"
	// 已移除 SSH 单元扫描，请手动配置 systemd 单元并使用 Agent 模式
	ErrMsg255ca1122356 = "已移除 SSH 单元扫描，请手动配置 systemd 单元并使用 Agent 模式"
	// 已移除 SSH 文件扫描，请手动配置路径并使用 Agent 模式
	ErrMsg36453c419629 = "已移除 SSH 文件扫描，请手动配置路径并使用 Agent 模式"
	// 已移除 SSH 日志流，请使用 Agent 日志流
	ErrMsgb399afd1b3b2 = "已移除 SSH 日志流，请使用 Agent 日志流"
	// 序列化直连配置失败
	ErrMsg2569b002d990 = "序列化直连配置失败"
	// 当前 IP 验证码请求过于频繁，请稍后重试
	ErrMsg26fa641b582b = "当前 IP 验证码请求过于频繁，请稍后重试"
	// 当前账号未绑定部门，无法管理用户
	ErrMsgc8caf91c1d57 = "当前账号未绑定部门，无法管理用户"
	// 成员不属于该项目
	ErrMsg461337ef3f89 = "成员不属于该项目"
	// 成员创建后查询失败
	ErrMsg1fe0209f952f = "成员创建后查询失败"
	// 成员更新后查询失败
	ErrMsg2940a3d4007c = "成员更新后查询失败"
	// 成员记录不存在
	ErrMsge7773625bf8b = "成员记录不存在"
	// 所属部门不存在
	ErrMsg94d63d947b0e = "所属部门不存在"
	// 抑制规则不存在
	ErrMsge4f20d76fd0d = "抑制规则不存在"
	// 排除正则表达式不合法
	ErrMsg9bbaf0815790 = "排除正则表达式不合法"
	// 数据库未初始化
	ErrMsg0ab7ea2d90bd = "数据库未初始化"
	// 数据源不存在
	ErrMsgaf3782e3e26f = "数据源不存在"
	// 数据源已停用
	ErrMsgfa357d889ce0 = "数据源已停用"
	// 文件上传失败
	ErrMsgf1b27684eaf6 = "文件上传失败"
	// 新密码不能与旧密码相同
	ErrMsg6ca55409b3c2 = "新密码不能与旧密码相同"
	// 无效的 effect，需为 NoSchedule、PreferNoSchedule 或 NoExecute
	ErrMsgdfbb6b0251fd = "无效的 effect，需为 NoSchedule、PreferNoSchedule 或 NoExecute"
	// 无权创建根部门
	ErrMsg685603d6807c = "无权创建根部门"
	// 无权在目标上级部门下创建
	ErrMsg099012ab5b6c = "无权在目标上级部门下创建"
	// 无权在该部门下创建用户
	ErrMsgd672e80435d4 = "无权在该部门下创建用户"
	// 无权将用户调整到目标部门
	ErrMsgc1305dfff708 = "无权将用户调整到目标部门"
	// 无权迁移到目标上级部门
	ErrMsgc23b85234e2a = "无权迁移到目标上级部门"
	// 无权限重启该 DaemonSet
	ErrMsg6e28e4e09c23 = "无权限重启该 DaemonSet"
	// 无权限重启该 Deployment
	ErrMsg4a3ba8680915 = "无权限重启该 Deployment"
	// 无权限重启该 StatefulSet
	ErrMsga0421725a51e = "无权限重启该 StatefulSet"
	// 无访问权限
	ErrMsgb47ec022e022 = "无访问权限"
	// 无该集群/命名空间操作权限
	ErrMsg1093cf9c0dc5 = "无该集群/命名空间操作权限"
	// 日志源不存在
	ErrMsg9d63941807e2 = "日志源不存在"
	// 日志源不属于当前服务器
	ErrMsgf528977ae67a = "日志源不属于当前服务器"
	// 日志源对应服务不存在
	ErrMsgce1b3b846df9 = "日志源对应服务不存在"
	// 旧密码不正确
	ErrMsg0767f3889e05 = "旧密码不正确"
	// 昵称不能为空
	ErrMsg702f83ff44f3 = "昵称不能为空"
	// 暂仅支持 type=prometheus
	ErrMsg480bba83b97b = "暂仅支持 type=prometheus"
	// 服务不存在
	ErrMsgac7e51a53391 = "服务不存在"
	// 服务器不在当前项目中
	ErrMsge75f9041a998 = "服务器不在当前项目中"
	// 服务器不存在
	ErrMsgb027c41d9692 = "服务器不存在"
	// 服务器不属于当前项目
	ErrMsg8394316b9c0a = "服务器不属于当前项目"
	// 服务器内部错误
	ErrMsg036549074827 = "服务器内部错误"
	// 服务器凭据未配置
	ErrMsgfeb33ee7c48c = "服务器凭据未配置"
	// 服务器分组不存在
	ErrMsg97c4c24a4cdf = "服务器分组不存在"
	// 服务器地址不能为空
	ErrMsgf2664ad99ec4 = "服务器地址不能为空"
	// 服务器已禁用
	ErrMsgb09d7fe54fc4 = "服务器已禁用"
	// 未找到匹配的 CR 资源类型
	ErrMsg1a5f8ce82917 = "未找到匹配的 CR 资源类型"
	// 未找到可用云账号（请先配置并同步云账号）
	ErrMsg38ffb6c1fcca = "未找到可用云账号（请先配置并同步云账号）"
	// 未找到指定容器；请填写正确的 container_name
	ErrMsg1a5aaa6cfa35 = "未找到指定容器；请填写正确的 container_name"
	// 未登录
	ErrMsgc1f658373079 = "未登录"
	// 未登录或登录已失效
	ErrMsg5bcb8895af9a = "未登录或登录已失效"
	// 权限不存在
	ErrMsg8140792dbe16 = "权限不存在"
	// 权限校验失败
	ErrMsg3968572e0ac2 = "权限校验失败"
	// 污点 effect 不能为空（NoSchedule / PreferNoSchedule / NoExecute）
	ErrMsg353da31eafa9 = "污点 effect 不能为空（NoSchedule / PreferNoSchedule / NoExecute）"
	// 污点 key 不能为空
	ErrMsgf545df0bc2cf = "污点 key 不能为空"
	// 注册密钥无效
	ErrMsg5fe5b6077279 = "注册密钥无效"
	// 注册申请不存在
	ErrMsg767a46d9ea99 = "注册申请不存在"
	// 源告警匹配条件不能为空
	ErrMsg5ca65f4bf6c2 = "源告警匹配条件不能为空"
	// 源告警正则表达式不合法
	ErrMsgc655c39dcf10 = "源告警正则表达式不合法"
	// 父节点不存在
	ErrMsgbe7758c9a279 = "父节点不存在"
	// 父节点不属于该项目
	ErrMsg5ddd2c9761c3 = "父节点不属于该项目"
	// 班次块不存在
	ErrMsgde63e900b907 = "班次块不存在"
	// 用户不存在
	ErrMsga82de572ccd7 = "用户不存在"
	// 用户仓储未初始化
	ErrMsgcc60c2c3c788 = "用户仓储未初始化"
	// 用户名不能为空
	ErrMsg390ccdec9f3f = "用户名不能为空"
	// 用户名已存在
	ErrMsgd1ee5409856b = "用户名已存在"
	// 用户名或密码错误
	ErrMsga9a6a83a7d22 = "用户名或密码错误"
	// 用户已被禁用
	ErrMsgfc81a2e790a8 = "用户已被禁用"
	// 用户未绑定邮箱
	ErrMsg55ce84d83c38 = "用户未绑定邮箱"
	// 登录已失效
	ErrMsg0759032719bf = "登录已失效"
	// 监控规则不存在
	ErrMsgdfcd891c9a94 = "监控规则不存在"
	// 目标告警匹配条件不能为空
	ErrMsg56384429bd38 = "目标告警匹配条件不能为空"
	// 目标告警正则表达式不合法
	ErrMsgc225462d5b41 = "目标告警正则表达式不合法"
	// 相同 AK 的云账号已存在
	ErrMsg74ea54455bb4 = "相同 AK 的云账号已存在"
	// 私钥不能为空
	ErrMsg6bd217c30983 = "私钥不能为空"
	// 缺少 Agent 令牌
	ErrMsga6a47518336c = "缺少 Agent 令牌"
	// 缺少 token 参数
	ErrMsgb6a3d51f342a = "缺少 token 参数"
	// 缺少密码凭据
	ErrMsg666b6d7186e5 = "缺少密码凭据"
	// 缺少或非法授权请求头
	ErrMsgf91112d5c51e = "缺少或非法授权请求头"
	// 缺少私钥凭据
	ErrMsg298c7d5f0d54 = "缺少私钥凭据"
	// 节点不存在
	ErrMsg7b4519294b96 = "节点不存在"
	// 节点名称不能为空
	ErrMsg215d21a8863c = "节点名称不能为空"
	// 获取 Pod 日志失败: 日志流为空
	ErrMsgf53aee0dab26 = "获取 Pod 日志失败: 日志流为空"
	// 获取 Pod 流日志失败: 日志流为空
	ErrMsgf59e8e01bf0d = "获取 Pod 流日志失败: 日志流为空"
	// 菜单不存在
	ErrMsg5e7ff218eaaa = "菜单不存在"
	// 角色不存在
	ErrMsg30dd0e6b0b53 = "角色不存在"
	// 订阅节点不存在
	ErrMsgb196d0c97d2f = "订阅节点不存在"
	// 认证类型不合法
	ErrMsgb6ada5b863ef = "认证类型不合法"
	// 该注册申请已审核，请勿重复操作
	ErrMsg2908bf4a32e1 = "该注册申请已审核，请勿重复操作"
	// 该用户名或邮箱已有待审核注册申请，请勿重复提交
	ErrMsgcd2bc0ed2797 = "该用户名或邮箱已有待审核注册申请，请勿重复提交"
	// 该用户已在项目中
	ErrMsga802e1b5e9e2 = "该用户已在项目中"
	// 该节点有子节点，请先删除子节点
	ErrMsgbc5e76aacb41 = "该节点有子节点，请先删除子节点"
	// 该部门下仍有关联用户，请先调整用户归属
	ErrMsga110c1191380 = "该部门下仍有关联用户，请先调整用户归属"
	// 请上传文件
	ErrMsg1e16294a3214 = "请上传文件"
	// 请先删除子菜单后再删除当前菜单
	ErrMsga70ebaf6959d = "请先删除子菜单后再删除当前菜单"
	// 请先删除子部门后再删除当前部门
	ErrMsgc22172530052 = "请先删除子部门后再删除当前部门"
	// 请选择需要批量操作的菜单
	ErrMsg83ecd70cfd99 = "请选择需要批量操作的菜单"
	// 读取上传文件失败
	ErrMsgc281d4dd7902 = "读取上传文件失败"
	// 资源不存在
	ErrMsg4aefbe3428ef = "资源不存在"
	// 资源清单不能为空
	ErrMsg01433598170d = "资源清单不能为空"
	// 路径不能为空
	ErrMsg72b2ecec3b64 = "路径不能为空"
	// 邮件服务未配置，暂时无法发送验证码
	ErrMsg1222f2978c2d = "邮件服务未配置，暂时无法发送验证码"
	// 邮件通道未配置收件人：请在邮件接收人或配置 JSON 中填写 to/recipients/emails；或由监控规则处理…
	ErrMsgc47e8ed41463 = "邮件通道未配置收件人：请在邮件接收人或配置 JSON 中填写 to/recipients/emails；或由监控规则处理人 assignee_emails 提供"
	// 邮件通道未配置：请检查全局 SMTP（mail 相关配置）是否启用
	ErrMsg71c5fe1e9994 = "邮件通道未配置：请检查全局 SMTP（mail 相关配置）是否启用"
	// 邮件通道至少需要配置一个收件人
	ErrMsgbb73fcfe4fb7 = "邮件通道至少需要配置一个收件人"
	// 邮件通道配置 JSON 解析失败，请检查 JSON 格式
	ErrMsg5b6514e2f558 = "邮件通道配置 JSON 解析失败，请检查 JSON 格式"
	// 邮箱已存在
	ErrMsgc4e82e866b26 = "邮箱已存在"
	// 邮箱已注册
	ErrMsg955ba8e30ca9 = "邮箱已注册"
	// 部分角色不存在
	ErrMsgbc90b8ad5f29 = "部分角色不存在"
	// 部门不存在
	ErrMsgbe049029249f = "部门不存在"
	// 部门名称和编码不能为空
	ErrMsga784a6e1674f = "部门名称和编码不能为空"
	// 部门编码已存在
	ErrMsgf5ccd8b73cf5 = "部门编码已存在"
	// 部门负责人不存在
	ErrMsgdccca82abd43 = "部门负责人不存在"
	// 钉钉应用会话模式需配置 appKey/appSecret/chatId
	ErrMsgf1768c17d51a = "钉钉应用会话模式需配置 appKey/appSecret/chatId"
	// 集群 ID 不合法
	ErrMsgba2a155d1253 = "集群 ID 不合法"
	// 集群已停用
	ErrMsgb0e556f1ccc5 = "集群已停用"
	// 静默不存在
	ErrMsgcef57a81fe5d = "静默不存在"
	// 非邮件通道必须填写 Webhook 地址
	ErrMsgaae2bd7c8c91 = "非邮件通道必须填写 Webhook 地址"
	// 项目 ID 与服务器归属不匹配
	ErrMsg39c3d6ae95e4 = "项目 ID 与服务器归属不匹配"
	// 项目不存在
	ErrMsgb2667460b6a9 = "项目不存在"
	// 验证码不能为空
	ErrMsgdb0b98dd46b0 = "验证码不能为空"
	// 验证码图片生成失败
	ErrMsg6f15f7c820be = "验证码图片生成失败"
	// 验证码场景不合法
	ErrMsga94172c66b0b = "验证码场景不合法"
	// 验证码已发送，请稍后再试
	ErrMsgd3ddd39f3ce3 = "验证码已发送，请稍后再试"
	// 验证码已过期或未发送，请重新获取
	ErrMsg5beab0d2bb4c = "验证码已过期或未发送，请重新获取"
	// 验证码已过期，请重新获取
	ErrMsgfa17d917b702 = "验证码已过期，请重新获取"
	// 验证码服务未就绪，请稍后重试
	ErrMsgaf4823214b6e = "验证码服务未就绪，请稍后重试"
	// 验证码邮件发送失败，请稍后重试
	ErrMsg52c1dc6bb947 = "验证码邮件发送失败，请稍后重试"
	// 验证码错误
	ErrMsg4f8238574720 = "验证码错误"
	// 验证码错误或已过期
	ErrMsg62fb79046777 = "验证码错误或已过期"
	// 验证码错误，请检查后重试
	ErrMsg002cf49fc664 = "验证码错误，请检查后重试"
	// 验证码长度必须大于 0
	ErrMsgb77c1b087c0b = "验证码长度必须大于 0"
)

// —— fmt.Sprintf 错误模板（配合 fmt.Sprintf 使用）——
const (
	// %s 语法错误: %v
	ErrFmt3664a9ad8a57 = "%s 语法错误: %v"
	// DaemonSet 重启失败: %v
	ErrFmt2ad7f69842e8 = "DaemonSet 重启失败: %v"
	// Deployment 扩缩容失败: %v
	ErrFmtdc6cb8eb04b7 = "Deployment 扩缩容失败: %v"
	// Deployment 重启失败: %v
	ErrFmt31dc761c382a = "Deployment 重启失败: %v"
	// K8s 心跳失败：%s
	ErrFmt5d75fe17f8ef = "K8s 心跳失败：%s"
	// Pod Exec 失败: %v
	ErrFmt1493bc1ea07a = "Pod Exec 失败: %v"
	// StatefulSet 扩缩容失败: %v
	ErrFmta91edbc01ba4 = "StatefulSet 扩缩容失败: %v"
	// StatefulSet 重启失败: %v
	ErrFmt83515735986c = "StatefulSet 重启失败: %v"
	// k8s 心跳失败: %v
	ErrFmt8648d0eaa652 = "k8s 心跳失败: %v"
	// k8s 连接失败: %v
	ErrFmtac130d1176b3 = "k8s 连接失败: %v"
	// limits 无效: %v
	ErrFmt81f1534a632d = "limits 无效: %v"
	// requests 无效: %v
	ErrFmte922f3829384 = "requests 无效: %v"
	// webhook 返回异常状态码: %d
	ErrFmtd0ae16233479 = "webhook 返回异常状态码: %d"
	// 上传文件到 Pod 失败: %v
	ErrFmt3a46eb93a86d = "上传文件到 Pod 失败: %v"
	// 从数据字典读取配置失败: %v
	ErrFmte5d845e17676 = "从数据字典读取配置失败: %v"
	// 保存上传文件失败: %v
	ErrFmt7e1c46be3a1e = "保存上传文件失败: %v"
	// 创建 exec 通道失败: %v
	ErrFmt1104a0b42122 = "创建 exec 通道失败: %v"
	// 创建临时文件失败: %v
	ErrFmt577d2b9acbc8 = "创建临时文件失败: %v"
	// 删除 %s 失败: %v
	ErrFmt32b88f9cc2e5 = "删除 %s 失败: %v"
	// 删除 CR 失败: %v
	ErrFmt8d45426dc121 = "删除 CR 失败: %v"
	// 删除 CRD 失败: %v
	ErrFmt233eb5aeea78 = "删除 CRD 失败: %v"
	// 删除 ConfigMap 失败: %v
	ErrFmt90788f2314d8 = "删除 ConfigMap 失败: %v"
	// 删除 Ingress 失败: %v
	ErrFmt0f7a7a13ca99 = "删除 Ingress 失败: %v"
	// 删除 IngressClass 失败: %v
	ErrFmt94bd2693979f = "删除 IngressClass 失败: %v"
	// 删除 NetworkPolicy 失败: %v
	ErrFmteaaed344b27b = "删除 NetworkPolicy 失败: %v"
	// 删除 Pod 失败: %v
	ErrFmt0b1419934c9b = "删除 Pod 失败: %v"
	// 删除 Pod 文件失败: %v
	ErrFmt94ef8ec4508c = "删除 Pod 文件失败: %v"
	// 删除 Secret 失败: %v
	ErrFmt931c82f11039 = "删除 Secret 失败: %v"
	// 删除 Service 失败: %v
	ErrFmtd89543e517f1 = "删除 Service 失败: %v"
	// 删除 ServiceAccount 失败: %v
	ErrFmtc85e19e5cfec = "删除 ServiceAccount 失败: %v"
	// 删除命名空间失败: %v
	ErrFmte323e75e3bb3 = "删除命名空间失败: %v"
	// 删除失败: %v
	ErrFmtd0193a5555ab = "删除失败: %v"
	// 处理上传文件失败: %v
	ErrFmt0881c0a79240 = "处理上传文件失败: %v"
	// 字典值过大（当前约 %d 字节，上限 %d 字节）
	ErrFmtd1b9788a27bb = "字典值过大（当前约 %d 字节，上限 %d 字节）"
	// 应用 YAML 失败: %v
	ErrFmt6d3ec85d0a18 = "应用 YAML 失败: %v"
	// 快捷创建 Pod 失败: %v
	ErrFmtdd4b4f8c2bd6 = "快捷创建 Pod 失败: %v"
	// 更新 CronJob suspend 失败: %v
	ErrFmt2dc26da7706f = "更新 CronJob suspend 失败: %v"
	// 更新 CronJob 资源配额失败: %v
	ErrFmtb9659e254f0c = "更新 CronJob 资源配额失败: %v"
	// 更新 DaemonSet 资源配额失败: %v
	ErrFmt4b5d3792ab7b = "更新 DaemonSet 资源配额失败: %v"
	// 更新 Deployment 资源配额失败: %v
	ErrFmtd272644d22c5 = "更新 Deployment 资源配额失败: %v"
	// 更新 Job 资源配额失败: %v
	ErrFmtcf70bcca046e = "更新 Job 资源配额失败: %v"
	// 更新 Node 污点失败: %v
	ErrFmtac67aae65acc = "更新 Node 污点失败: %v"
	// 更新 Node 调度状态失败: %v
	ErrFmt6f761b85c92e = "更新 Node 调度状态失败: %v"
	// 更新 Pod 镜像失败: %v
	ErrFmta70db4595e64 = "更新 Pod 镜像失败: %v"
	// 更新 StatefulSet 资源配额失败: %v
	ErrFmt6cf03553be44 = "更新 StatefulSet 资源配额失败: %v"
	// 正则表达式错误 [%s]: %v
	ErrFmtc37306c826dc = "正则表达式错误 [%s]: %v"
	// 生成kubeconfig失败: %v
	ErrFmt92e759c1fa53 = "生成kubeconfig失败: %v"
	// 等待 Pod 删除失败: %v
	ErrFmtbca6d25dcb71 = "等待 Pod 删除失败: %v"
	// 编辑并重建 Pod 失败: %v
	ErrFmt57c6f6e7ba6b = "编辑并重建 Pod 失败: %v"
	// 获取 CR 列表失败: %v
	ErrFmt0f0376756412 = "获取 CR 列表失败: %v"
	// 获取 CR 详情失败: %v
	ErrFmt076435509f8c = "获取 CR 详情失败: %v"
	// 获取 CRD 列表失败: %v
	ErrFmtcf9172f6e822 = "获取 CRD 列表失败: %v"
	// 获取 CRD 详情失败: %v
	ErrFmt48ec32249aa2 = "获取 CRD 详情失败: %v"
	// 获取 ClusterRole 列表失败: %v
	ErrFmt5f9235a5cff6 = "获取 ClusterRole 列表失败: %v"
	// 获取 ClusterRoleBinding 列表失败: %v
	ErrFmt05b921e3a64d = "获取 ClusterRoleBinding 列表失败: %v"
	// 获取 ConfigMap 失败: %v
	ErrFmtbfbd3990363e = "获取 ConfigMap 失败: %v"
	// 获取 ConfigMaps 失败: %v
	ErrFmtf8a8a793f3c6 = "获取 ConfigMaps 失败: %v"
	// 获取 CronJob 关联 Jobs 失败: %v
	ErrFmtc13b046a7597 = "获取 CronJob 关联 Jobs 失败: %v"
	// 获取 CronJob 失败: %v
	ErrFmt687b79e3dfdb = "获取 CronJob 失败: %v"
	// 获取 CronJobs 失败: %v
	ErrFmt336d54b211b0 = "获取 CronJobs 失败: %v"
	// 获取 DaemonSet 失败: %v
	ErrFmt960ced5a2f6f = "获取 DaemonSet 失败: %v"
	// 获取 DaemonSets 失败: %v
	ErrFmt22f7c7b69366 = "获取 DaemonSets 失败: %v"
	// 获取 Deployment 失败: %v
	ErrFmta3018a66177e = "获取 Deployment 失败: %v"
	// 获取 Deployments 失败: %v
	ErrFmt78bb8313c519 = "获取 Deployments 失败: %v"
	// 获取 Events 失败: %v
	ErrFmtd678ffdd4e0f = "获取 Events 失败: %v"
	// 获取 Ingress 列表失败: %v
	ErrFmt7f0818fd6f52 = "获取 Ingress 列表失败: %v"
	// 获取 Ingress 详情失败: %v
	ErrFmtd0e7b9970841 = "获取 Ingress 详情失败: %v"
	// 获取 IngressClass 列表失败: %v
	ErrFmt6c250f47f18b = "获取 IngressClass 列表失败: %v"
	// 获取 IngressClass 详情失败: %v
	ErrFmt829d798aa9fb = "获取 IngressClass 详情失败: %v"
	// 获取 Job 失败: %v
	ErrFmt1a7e7f82dbdc = "获取 Job 失败: %v"
	// 获取 Jobs 失败: %v
	ErrFmt9987a1977622 = "获取 Jobs 失败: %v"
	// 获取 NetworkPolicy 列表失败: %v
	ErrFmte5f4df2bc9c2 = "获取 NetworkPolicy 列表失败: %v"
	// 获取 NetworkPolicy 详情失败: %v
	ErrFmtd28ea35ac553 = "获取 NetworkPolicy 详情失败: %v"
	// 获取 Node 列表失败: %v
	ErrFmt6af6d441fc65 = "获取 Node 列表失败: %v"
	// 获取 Node 失败: %v
	ErrFmta293b9a12001 = "获取 Node 失败: %v"
	// 获取 Node 详情失败: %v
	ErrFmt743663002376 = "获取 Node 详情失败: %v"
	// 获取 PV 列表失败: %v
	ErrFmtea2113ac1281 = "获取 PV 列表失败: %v"
	// 获取 PVC 列表失败: %v
	ErrFmt3503e02d7ad4 = "获取 PVC 列表失败: %v"
	// 获取 Pod 事件失败: %v
	ErrFmtbf8c73dd9a9e = "获取 Pod 事件失败: %v"
	// 获取 Pod 失败: %v
	ErrFmte69166988489 = "获取 Pod 失败: %v"
	// 获取 Pod 文件列表失败: %v
	ErrFmt85e9441edd38 = "获取 Pod 文件列表失败: %v"
	// 获取 Pod 日志失败: %v
	ErrFmtd868fafc39ca = "获取 Pod 日志失败: %v"
	// 获取 Pod 流日志失败: %v
	ErrFmt43046963a139 = "获取 Pod 流日志失败: %v"
	// 获取 Pod 详情失败: %v
	ErrFmtc52b9130d74c = "获取 Pod 详情失败: %v"
	// 获取 Role 列表失败: %v
	ErrFmt951c7ad77bce = "获取 Role 列表失败: %v"
	// 获取 RoleBinding 列表失败: %v
	ErrFmt3489b1e268aa = "获取 RoleBinding 列表失败: %v"
	// 获取 Secret 失败: %v
	ErrFmtb450a71bd00f = "获取 Secret 失败: %v"
	// 获取 Secrets 失败: %v
	ErrFmta2182ce2ddcd = "获取 Secrets 失败: %v"
	// 获取 Service 列表失败: %v
	ErrFmt38cc1640ac12 = "获取 Service 列表失败: %v"
	// 获取 Service 详情失败: %v
	ErrFmt42eaecf6d979 = "获取 Service 详情失败: %v"
	// 获取 ServiceAccount 列表失败: %v
	ErrFmtf425dc675e3d = "获取 ServiceAccount 列表失败: %v"
	// 获取 ServiceAccount 详情失败: %v
	ErrFmtc9a14a0f8fc2 = "获取 ServiceAccount 详情失败: %v"
	// 获取 StatefulSet 失败: %v
	ErrFmt70dba6fa52bd = "获取 StatefulSet 失败: %v"
	// 获取 StatefulSets 失败: %v
	ErrFmt3bef5bb60df3 = "获取 StatefulSets 失败: %v"
	// 获取 StorageClass 列表失败: %v
	ErrFmt12c6283be648 = "获取 StorageClass 列表失败: %v"
	// 获取 ingress-nginx Pods 失败: %v
	ErrFmt0cbe9766f7af = "获取 ingress-nginx Pods 失败: %v"
	// 获取关联 Pods 失败: %v
	ErrFmt3ab38ee441a3 = "获取关联 Pods 失败: %v"
	// 获取命名空间失败: %v
	ErrFmt8d60c2040f20 = "获取命名空间失败: %v"
	// 获取命名空间详情失败: %v
	ErrFmt059d07c698fe = "获取命名空间详情失败: %v"
	// 获取组件状态失败: %v
	ErrFmt559cb56d5b9d = "获取组件状态失败: %v"
	// 获取详情失败: %v
	ErrFmt0aa6043acdf6 = "获取详情失败: %v"
	// 解析 CR 类型失败: %v
	ErrFmt2b30d4949c98 = "解析 CR 类型失败: %v"
	// 解析 kubeconfig 失败: %v
	ErrFmtd7f0c3fe8497 = "解析 kubeconfig 失败: %v"
	// 触发 CronJob 创建 Job 失败: %v
	ErrFmt25f9e144a662 = "触发 CronJob 创建 Job 失败: %v"
	// 请求过于频繁，请 %d 秒后重试
	ErrFmte5ea7331dbac = "请求过于频繁，请 %d 秒后重试"
	// 读取 Pod 文件失败: %v
	ErrFmt2b7dfae8ff2c = "读取 Pod 文件失败: %v"
	// 读取 Pod 日志失败: %v
	ErrFmt8e15ae24a3a1 = "读取 Pod 日志失败: %v"
	// 重启 Pod 失败: %v
	ErrFmtf802a81a6173 = "重启 Pod 失败: %v"
	// 重新执行 Job 失败: %v
	ErrFmt2abaeffc289e = "重新执行 Job 失败: %v"
)
