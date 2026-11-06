package responses

import (
	"context"
	"errors"
	"github.com/elcomware/edupilot/platform/access"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func RPCError(err error) error {
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return status.FromContextError(err).Err()
	case errors.Is(err, access.ErrUnauthenticated):
		return status.Error(codes.Unauthenticated, "Invalid or expired session.")
	case errors.Is(err, access.ErrForbidden):
		return status.Error(codes.PermissionDenied, "Access denied for this school.")
	default:
		return status.Error(codes.Internal, "Unable to process request.")
	}
}
