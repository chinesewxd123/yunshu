package store

func AccessTokenKey(tokenID string) string {
	return "auth:access_token:" + tokenID
}
