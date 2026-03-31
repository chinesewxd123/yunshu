package handler

import "go-permission-system/internal/service"

type HealthData struct {
	Name string `json:"name" example:"permission-system"`
	Env  string `json:"env" example:"dev"`
}

type MessageData struct {
	Message string `json:"message" example:"success"`
}

type UserPageData struct {
	List     []service.UserDetailResponse `json:"list"`
	Total    int64                        `json:"total" example:"1"`
	Page     int                          `json:"page" example:"1"`
	PageSize int                          `json:"page_size" example:"10"`
}

type RolePageData struct {
	List     []service.RoleItem `json:"list"`
	Total    int64              `json:"total" example:"1"`
	Page     int                `json:"page" example:"1"`
	PageSize int                `json:"page_size" example:"10"`
}

type PermissionPageData struct {
	List     []service.PermissionItem `json:"list"`
	Total    int64                    `json:"total" example:"1"`
	Page     int                      `json:"page" example:"1"`
	PageSize int                      `json:"page_size" example:"10"`
}
