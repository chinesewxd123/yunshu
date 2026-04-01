package service

import (
	"context"
	"errors"

	"go-permission-system/internal/model"
	"go-permission-system/internal/repository"
)

type MenuService struct {
	menuRepo *repository.MenuRepository
}

func NewMenuService(menuRepo *repository.MenuRepository) *MenuService {
	return &MenuService{menuRepo: menuRepo}
}

type MenuCreatePayload struct {
	ParentID  *uint  `json:"parent_id"`
	Path      string `json:"path"`
	Name      string `json:"name" binding:"required,max=64"`
	Icon      string `json:"icon"`
	Sort      int    `json:"sort"`
	Hidden    bool   `json:"hidden"`
	Component string `json:"component"`
	Redirect  string `json:"redirect"`
	Status    int    `json:"status" binding:"required,oneof=0 1"`
}

type MenuUpdatePayload struct {
	ParentID  *uint  `json:"parent_id"`
	Path      string `json:"path"`
	Name      string `json:"name" binding:"omitempty,max=64"`
	Icon      string `json:"icon"`
	Sort      int    `json:"sort"`
	Hidden    bool   `json:"hidden"`
	Component string `json:"component"`
	Redirect  string `json:"redirect"`
	Status    int    `json:"status" binding:"omitempty,oneof=0 1"`
}

func (s *MenuService) Tree(ctx context.Context) ([]model.Menu, error) {
	return s.menuRepo.Tree(ctx)
}

func (s *MenuService) Create(ctx context.Context, payload MenuCreatePayload) (*model.Menu, error) {
	menu := model.Menu{
		ParentID:  payload.ParentID,
		Path:      payload.Path,
		Name:      payload.Name,
		Icon:      payload.Icon,
		Sort:      payload.Sort,
		Hidden:    payload.Hidden,
		Component: payload.Component,
		Redirect:  payload.Redirect,
		Status:    payload.Status,
	}
	if err := s.menuRepo.Create(ctx, &menu); err != nil {
		return nil, err
	}
	return &menu, nil
}

func (s *MenuService) Update(ctx context.Context, id uint, payload MenuUpdatePayload) (*model.Menu, error) {
	menu, err := s.menuRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if payload.Name != "" {
		menu.Name = payload.Name
	}
	if payload.Path != "" {
		menu.Path = payload.Path
	}
	menu.Icon = payload.Icon
	menu.Sort = payload.Sort
	menu.Hidden = payload.Hidden
	menu.Component = payload.Component
	menu.Redirect = payload.Redirect
	if payload.Status != 0 || id == menu.ID {
		menu.Status = payload.Status
	}
	if payload.ParentID != nil {
		menu.ParentID = payload.ParentID
	}
	if err := s.menuRepo.Update(ctx, menu); err != nil {
		return nil, err
	}
	return menu, nil
}

func (s *MenuService) Delete(ctx context.Context, id uint) error {
	count, err := s.menuRepo.CountChildren(ctx, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New("please delete child menus first")
	}
	return s.menuRepo.Delete(ctx, id)
}
