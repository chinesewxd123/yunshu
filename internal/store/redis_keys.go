package store

import "strings"

func AccessTokenKey(tokenID string) string {
	return "auth:access_token:" + tokenID
}

func EmailCodeKey(scene, email string) string {
	return "auth:email_code:" + strings.TrimSpace(scene) + ":" + normalizeEmailKey(email)
}

func EmailCodeCooldownKey(scene, email string) string {
	return "auth:email_code_cooldown:" + strings.TrimSpace(scene) + ":" + normalizeEmailKey(email)
}

func PasswordLoginCodeKey(username string) string {
	return "auth:password_login_code:" + strings.TrimSpace(username)
}

func PasswordLoginCodeCooldownKey(username string) string {
	return "auth:password_login_code_cooldown:" + strings.TrimSpace(username)
}

func normalizeEmailKey(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
