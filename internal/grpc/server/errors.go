package server

import (
	"errors"

	"yunshu/internal/pkg/apperror"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func toStatusErr(err error) error {
	if err == nil {
		return nil
	}
	var appErr *apperror.AppError
	if errors.As(err, &appErr) {
		switch appErr.StatusCode {
		case 400:
			return status.Error(codes.InvalidArgument, appErr.Message)
		case 401:
			return status.Error(codes.Unauthenticated, appErr.Message)
		case 403:
			return status.Error(codes.PermissionDenied, appErr.Message)
		case 404:
			return status.Error(codes.NotFound, appErr.Message)
		case 409:
			return status.Error(codes.AlreadyExists, appErr.Message)
		default:
			return status.Error(codes.Internal, appErr.Message)
		}
	}
	return status.Error(codes.Internal, err.Error())
}
