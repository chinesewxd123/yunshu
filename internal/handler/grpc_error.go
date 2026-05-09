package handler

import (
	"yunshu/internal/pkg/constants"

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
		return constants.ErrBadRequestWithMsg(st.Message())
	case codes.Unauthenticated:
		return constants.ErrUnauthorizedWithMsg(st.Message())
	case codes.PermissionDenied:
		return constants.ErrForbiddenWithMsg(st.Message())
	case codes.NotFound:
		return constants.ErrNotFoundWithMsg(st.Message())
	case codes.AlreadyExists:
		return constants.ErrConflictWithMsg(st.Message())
	default:
		return constants.ErrInternalWithMsg(st.Message())
	}
}
