package grpc

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/Belphemur/SuperSubtitles/v2/internal/apperrors"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func toStatusError(fallbackMessage string, err error) error {
	if err == nil {
		return nil
	}

	var bindable apperrors.GRPCBindableError
	if errors.As(err, &bindable) {
		return statusForBindableError(bindable.GRPCCode(), err.Error(), bindable.HTTPStatusCode())
	}

	return status.Errorf(codes.Internal, "%s: %v", fallbackMessage, err)
}

func statusForBindableError(code codes.Code, message string, httpStatus int) error {
	st := status.New(code, message)
	if httpStatus <= 0 {
		return st.Err()
	}

	reason := "HTTP_STATUS_" + strconv.Itoa(httpStatus)
	if httpStatus == http.StatusUnprocessableEntity {
		reason = "UNPROCESSABLE_ENTITY"
	}

	withDetails, err := st.WithDetails(&errdetails.ErrorInfo{
		Reason: reason,
		Metadata: map[string]string{
			"http_status": strconv.Itoa(httpStatus),
		},
	})
	if err != nil {
		return status.Errorf(code, "%s (http_status=%d)", message, httpStatus)
	}

	return withDetails.Err()
}
