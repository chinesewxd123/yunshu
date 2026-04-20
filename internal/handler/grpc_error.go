package handler

import (
	"yunshu/internal/pkg/apperror"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func grpcToAppError(err error) error {
	if err == nil {
		return nil
	}
	st, ok := status.FromError(err)
	if !ok {
		return err
	}
	switch st.Code() {
	case codes.InvalidArgument:
		return apperror.BadRequest(st.Message())
	case codes.Unauthenticated:
		return apperror.Unauthorized(st.Message())
	case codes.PermissionDenied:
		return apperror.Forbidden(st.Message())
	case codes.NotFound:
		return apperror.NotFound(st.Message())
	case codes.AlreadyExists:
		return apperror.Conflict(st.Message())
	default:
		return apperror.Internal(st.Message())
	}
}
