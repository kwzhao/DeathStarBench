package tracing

import (
	"context"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

// ChainUnaryServerInterceptors chains multiple unary server interceptors into a single interceptor.
func ChainUnaryServerInterceptors(interceptors ...grpc.UnaryServerInterceptor) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Nested handler function to call the interceptors in order
		chainedHandler := handler
		for i := len(interceptors) - 1; i >= 0; i-- {
			interceptor := interceptors[i]
			next := chainedHandler
			chainedHandler = func(ctx context.Context, req interface{}) (interface{}, error) {
				return interceptor(ctx, req, info, next)
			}
		}
		return chainedHandler(ctx, req)
	}
}

// SizeTaggingUnaryServerInterceptor tags the OpenTracing span with the request and response sizes.
func SizeTaggingUnaryServerInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	span := opentracing.SpanFromContext(ctx)
	if span != nil {
		if reqProto, ok := req.(proto.Message); ok {
			reqSize := proto.Size(reqProto)
			span.SetTag("grpc.request.size", reqSize)
			log.Info().Msgf("Request size for %s: %d bytes", info.FullMethod, reqSize)
		} else {
			log.Warn().Msgf("Request for method %s is not a proto.Message", info.FullMethod)
			span.SetTag("grpc.request.size", -1)
		}
	}
	resp, err := handler(ctx, req)
	if span != nil {
		if err == nil {
			if respProto, ok := resp.(proto.Message); ok {
				respSize := proto.Size(respProto)
				span.SetTag("grpc.response.size", respSize)
				log.Info().Msgf("Response size for %s: %d bytes", info.FullMethod, respSize)
			} else {
				log.Warn().Msgf("Response for method %s is not a proto.Message", info.FullMethod)
				span.SetTag("grpc.response.size", -1)
			}
		}
	}
	return resp, err
}

// ChainUnaryClientInterceptors chains multiple unary client interceptors into a single interceptor
func ChainUnaryClientInterceptors(interceptors ...grpc.UnaryClientInterceptor) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		chainedInvoker := invoker
		for i := len(interceptors) - 1; i >= 0; i-- {
			interceptor := interceptors[i]
			next := chainedInvoker
			chainedInvoker = func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
				return interceptor(ctx, method, req, reply, cc, next, opts...)
			}
		}
		return chainedInvoker(ctx, method, req, reply, cc, opts...)
	}
}

// SizeTaggingUnaryClientInterceptor tags the OpenTracing span with the request and response sizes
func SizeTaggingUnaryClientInterceptor(
	ctx context.Context,
	method string,
	req, reply interface{},
	cc *grpc.ClientConn,
	invoker grpc.UnaryInvoker,
	opts ...grpc.CallOption,
) error {
	span := opentracing.SpanFromContext(ctx)
	if span != nil {
		if reqProto, ok := req.(proto.Message); ok {
			reqSize := proto.Size(reqProto)
			span.SetTag("grpc.request.size", reqSize)
			log.Info().Msgf("Request size for %s: %d bytes", method, reqSize)
		} else {
			log.Warn().Msgf("Request for method %s is not a proto.Message", method)
			span.SetTag("grpc.request.size", "unknown")
		}
	}
	err := invoker(ctx, method, req, reply, cc, opts...)
	if span != nil {
		if err == nil {
			if replyProto, ok := reply.(proto.Message); ok {
				respSize := proto.Size(replyProto)
				span.SetTag("grpc.response.size", respSize)
				log.Info().Msgf("Response size for %s: %d bytes", method, respSize)
			} else {
				log.Warn().Msgf("Response for method %s is not a proto.Message", method)
				span.SetTag("grpc.response.size", "unknown")
			}
		}
	}
	return err
}
