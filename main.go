package main

// @title Permission System API
// @version 1.0
// @description Permission management system built with Gin, MySQL, Redis, Casbin, Cobra and slog.
// @description Use the login endpoint first, then put `Bearer <token>` into the Authorization header.
// @BasePath /
// @schemes http https
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter the JWT token with the `Bearer ` prefix, for example: Bearer eyJhbGciOi...
import "go-permission-system/cmd"

func main() {
	cmd.Execute()
}
