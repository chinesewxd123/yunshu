package service

import (
	"errors"

	"yunshu/internal/pkg/apperror"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/service/svcerr"

	"gorm.io/gorm"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

// k8sFail 统一处理 K8s Service 层错误：保留 AppError；映射 apiserver 状态码；其余走 svcerr.Pass。
func k8sFail(component, operation string, err error, attrs ...any) error {
	if err == nil {
		return nil
	}
	if _, ok := apperror.IsAppError(err); ok {
		return err
	}
	if mapped := k8sMapAPIError(err); mapped != err {
		if _, ok := apperror.IsAppError(mapped); ok {
			return mapped
		}
	}
	return svcerr.Pass(component, operation, err, attrs...)
}

// k8sRepoErr 将仓储层错误转为业务错误（未找到 → 10004）。
func k8sRepoErr(component, operation string, err error, attrs ...any) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return constants.ErrNotFound
	}
	return svcerr.Pass(component, operation, err, attrs...)
}

func k8sMapAPIError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case apierrors.IsNotFound(err):
		return constants.ErrNotFound
	case apierrors.IsForbidden(err):
		return constants.ErrForbidden
	case apierrors.IsConflict(err):
		return constants.ErrConflict
	case apierrors.IsUnauthorized(err):
		return constants.ErrUnauthorized
	case apierrors.IsInvalid(err), apierrors.IsBadRequest(err):
		msg := string(apierrors.ReasonForError(err))
		if msg == "" {
			msg = err.Error()
		}
		return constants.ErrBadRequestWithMsg(msg)
	default:
		return err
	}
}
