package http2

import "strings"

// google/rpc Code values (code.proto).
const (
	grpcOK                 = 0
	grpcCancelled          = 1
	grpcUnknown            = 2
	grpcInvalidArgument    = 3
	grpcDeadlineExceeded   = 4
	grpcNotFound           = 5
	grpcAlreadyExists      = 6
	grpcPermissionDenied   = 7
	grpcResourceExhausted  = 8
	grpcFailedPrecondition = 9
	grpcAborted            = 10
	grpcOutOfRange         = 11
	grpcUnimplemented      = 12
	grpcInternal           = 13
	grpcUnavailable        = 14
	grpcDataLoss           = 15
	grpcUnauthenticated    = 16
)

// grpcStatusHTTP maps google/rpc Code to the HTTP status used for status_class.
// Matches google/rpc/code.proto and grpc gateway HTTPStatusFromCode.
func grpcStatusHTTP(code int) int {
	switch code {
	case grpcOK:
		return 200
	case grpcCancelled:
		return 499
	case grpcUnknown:
		return 500
	case grpcInvalidArgument:
		return 400
	case grpcDeadlineExceeded:
		return 504
	case grpcNotFound:
		return 404
	case grpcAlreadyExists:
		return 409
	case grpcPermissionDenied:
		return 403
	case grpcResourceExhausted:
		return 429
	case grpcFailedPrecondition:
		return 400
	case grpcAborted:
		return 409
	case grpcOutOfRange:
		return 400
	case grpcUnimplemented:
		return 501
	case grpcInternal:
		return 500
	case grpcUnavailable:
		return 503
	case grpcDataLoss:
		return 500
	case grpcUnauthenticated:
		return 401
	default:
		return 500
	}
}

func isGRPCContentType(ct string) bool {
	return ct == "application/grpc" || strings.HasPrefix(ct, "application/grpc+")
}
