package middleware

import (
	"context"

	"google.golang.org/grpc"
)

// GRPCErrorUnaryWrapper implements wrapper of unary handler with session in context
func GRPCErrorUnaryWrapper(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
	resp, err = handler(ctx, req)
	err = prepareHandlerError(err)
	return resp, err
}

// GRPCErrorStreamWrapper implements wrapper of stream handler with error processing
func GRPCErrorStreamWrapper(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	err := handler(srv, ss)
	err = prepareHandlerError(err)
	return err
}

func prepareHandlerError(err error) error {
	// if err == nil {
	// 	return nil
	// }
	// switch err.(type) {
	// case interface{ GRPCStatus() *status.Status }:
	// // case *protocol.ValidateError:
	// // 	err = grpcgw.NewHTTPError(int(codes.FailedPrecondition), err.Error())
	// default:
	// 	switch {
	// 	case errors.Is(err, sql.ErrNoRows):
	// 		errMsg := err.Error()
	// 		if strings.HasSuffix(errMsg, sql.ErrNoRows.Error()) {
	// 			errMsg = strings.TrimSuffix(err.Error(), sql.ErrNoRows.Error()) + "Not Found"
	// 		}
	// 		return grpcgw.NewHTTPError(int(codes.NotFound), errMsg)
	// 		// case errors.Is(err, acl.ErrNoPermissions):
	// 		// 	return grpcgw.NewHTTPError(int(codes.PermissionDenied), err.Error())
	// 	}
	// }
	return err
}
