// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.21.12
// source: dapr/proto/scheduler/v1/scheduler_callback.proto

package scheduler

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// SchedulerCallbackClient is the client API for SchedulerCallback service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type SchedulerCallbackClient interface {
	// Callback RPC for job schedule being at 'trigger' time
	TriggerJob(ctx context.Context, in *TriggerJobRequest, opts ...grpc.CallOption) (*TriggerJobResponse, error)
}

type schedulerCallbackClient struct {
	cc grpc.ClientConnInterface
}

func NewSchedulerCallbackClient(cc grpc.ClientConnInterface) SchedulerCallbackClient {
	return &schedulerCallbackClient{cc}
}

func (c *schedulerCallbackClient) TriggerJob(ctx context.Context, in *TriggerJobRequest, opts ...grpc.CallOption) (*TriggerJobResponse, error) {
	out := new(TriggerJobResponse)
	err := c.cc.Invoke(ctx, "/dapr.proto.scheduler.v1.SchedulerCallback/TriggerJob", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// SchedulerCallbackServer is the server API for SchedulerCallback service.
// All implementations should embed UnimplementedSchedulerCallbackServer
// for forward compatibility
type SchedulerCallbackServer interface {
	// Callback RPC for job schedule being at 'trigger' time
	TriggerJob(context.Context, *TriggerJobRequest) (*TriggerJobResponse, error)
}

// UnimplementedSchedulerCallbackServer should be embedded to have forward compatible implementations.
type UnimplementedSchedulerCallbackServer struct {
}

func (UnimplementedSchedulerCallbackServer) TriggerJob(context.Context, *TriggerJobRequest) (*TriggerJobResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method TriggerJob not implemented")
}

// UnsafeSchedulerCallbackServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to SchedulerCallbackServer will
// result in compilation errors.
type UnsafeSchedulerCallbackServer interface {
	mustEmbedUnimplementedSchedulerCallbackServer()
}

func RegisterSchedulerCallbackServer(s grpc.ServiceRegistrar, srv SchedulerCallbackServer) {
	s.RegisterService(&SchedulerCallback_ServiceDesc, srv)
}

func _SchedulerCallback_TriggerJob_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(TriggerJobRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SchedulerCallbackServer).TriggerJob(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/dapr.proto.scheduler.v1.SchedulerCallback/TriggerJob",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SchedulerCallbackServer).TriggerJob(ctx, req.(*TriggerJobRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// SchedulerCallback_ServiceDesc is the grpc.ServiceDesc for SchedulerCallback service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var SchedulerCallback_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "dapr.proto.scheduler.v1.SchedulerCallback",
	HandlerType: (*SchedulerCallbackServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "TriggerJob",
			Handler:    _SchedulerCallback_TriggerJob_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "dapr/proto/scheduler/v1/scheduler_callback.proto",
}
