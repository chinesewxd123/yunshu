package main

// permission-system 后端可执行入口：将 CLI 委托给 cmd 包（server / migrate / log-agent 等）。
//
// @title YunShu CMDB API
// @version 1.0
// @description YunShu CMDB is an operations CMDB console built with Gin, MySQL, Redis, SMTP mail delivery, Casbin, Cobra and slog.
// @description Request an email verification code first, then login with email code or register a new account by email. Username/password login remains available as an admin fallback.
// @BasePath /
// @schemes http https
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter the JWT token with the `Bearer ` prefix, for example: Bearer eyJhbGciOi...
import "yunshu/cmd"

func main() {
	cmd.Execute()
}
