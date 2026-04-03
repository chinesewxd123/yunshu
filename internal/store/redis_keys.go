package store

import "strings"

// AccessTokenKey 访问令牌缓存键
func AccessTokenKey(tokenID string) string {
	return "auth:access_token:" + tokenID
}

// EmailCodeKey 邮箱验证码缓存键
func EmailCodeKey(scene, email string) string {
	return "auth:email_code:" + strings.TrimSpace(scene) + ":" + normalizeEmailKey(email)
}

// EmailCodeCooldownKey 邮箱验证码冷却缓存键
func EmailCodeCooldownKey(scene, email string) string {
	return "auth:email_code_cooldown:" + strings.TrimSpace(scene) + ":" + normalizeEmailKey(email)
}

// PasswordLoginCodeKey 密码登录验证码缓存键
func PasswordLoginCodeKey(username string) string {
	return "auth:password_login_code:" + strings.TrimSpace(username)
}

// PasswordLoginCodeCooldownKey 密码登录验证码冷却缓存键
func PasswordLoginCodeCooldownKey(username string) string {
	return "auth:password_login_code_cooldown:" + strings.TrimSpace(username)
}

// normalizeEmailKey 邮箱缓存键归一化函数
func normalizeEmailKey(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// EmailSendIPKey key for tracking number of email sends from an IP
func EmailSendIPKey(ip string) string {
	return "auth:email_send_ip:" + strings.TrimSpace(ip)
}

// BanIPKey temporary ban key for IPs
func BanIPKey(ip string) string {
	return "ban:ip:" + strings.TrimSpace(ip)
}
