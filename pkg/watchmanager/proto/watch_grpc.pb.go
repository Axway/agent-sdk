package proto

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

// WatchServiceClient is the client API for WatchService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type WatchServiceClient interface {
	CreateWatch(ctx context.Context, in *Request, opts ...grpc.CallOption) (WatchService_CreateWatchClient, error)
	SubscribeToTopic(ctx context.Context, in *Topic, opts ...grpc.CallOption) (WatchService_SubscribeToTopicClient, error)
}

type watchServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewWatchServiceClient(cc grpc.ClientConnInterface) WatchServiceClient {
	return &watchServiceClient{cc}
}

func (c *watchServiceClient) CreateWatch(ctx context.Context, in *Request, opts ...grpc.CallOption) (WatchService_CreateWatchClient, error) {
	stream, err := c.cc.NewStream(ctx, &WatchService_ServiceDesc.Streams[0], "/apis.v1.WatchService/CreateWatch", opts...)
	if err != nil {
		return nil, err
	}
	x := &watchServiceCreateWatchClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type WatchService_CreateWatchClient interface {
	Recv() (*Event, error)
	grpc.ClientStream
}

type watchServiceCreateWatchClient struct {
	grpc.ClientStream
}

func (x *watchServiceCreateWatchClient) Recv() (*Event, error) {
	m := new(Event)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *watchServiceClient) SubscribeToTopic(ctx context.Context, in *Topic, opts ...grpc.CallOption) (WatchService_SubscribeToTopicClient, error) {
	stream, err := c.cc.NewStream(ctx, &WatchService_ServiceDesc.Streams[1], "/apis.v1.WatchService/SubscribeToTopic", opts...)
	if err != nil {
		return nil, err
	}
	x := &watchServiceSubscribeToTopicClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type WatchService_SubscribeToTopicClient interface {
	Recv() (*Event, error)
	grpc.ClientStream
}

type watchServiceSubscribeToTopicClient struct {
	grpc.ClientStream
}

func (x *watchServiceSubscribeToTopicClient) Recv() (*Event, error) {
	m := new(Event)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// WatchServiceServer is the server API for WatchService service.
// All implementations must embed UnimplementedWatchServiceServer
// for forward compatibility
type WatchServiceServer interface {
	CreateWatch(*Request, WatchService_CreateWatchServer) error
	SubscribeToTopic(*Topic, WatchService_SubscribeToTopicServer) error
	mustEmbedUnimplementedWatchServiceServer()
}

// UnimplementedWatchServiceServer must be embedded to have forward compatible implementations.
type UnimplementedWatchServiceServer struct {
}

func (UnimplementedWatchServiceServer) CreateWatch(*Request, WatchService_CreateWatchServer) error {
	return status.Errorf(codes.Unimplemented, "method CreateWatch not implemented")
}
func (UnimplementedWatchServiceServer) SubscribeToTopic(*Topic, WatchService_SubscribeToTopicServer) error {
	return status.Errorf(codes.Unimplemented, "method SubscribeToTopic not implemented")
}
func (UnimplementedWatchServiceServer) mustEmbedUnimplementedWatchServiceServer() {}

// UnsafeWatchServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to WatchServiceServer will
// result in compilation errors.
type UnsafeWatchServiceServer interface {
	mustEmbedUnimplementedWatchServiceServer()
}

func RegisterWatchServiceServer(s grpc.ServiceRegistrar, srv WatchServiceServer) {
	s.RegisterService(&WatchService_ServiceDesc, srv)
}

func _WatchService_CreateWatch_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(Request)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(WatchServiceServer).CreateWatch(m, &watchServiceCreateWatchServer{stream})
}

type WatchService_CreateWatchServer interface {
	Send(*Event) error
	grpc.ServerStream
}

type watchServiceCreateWatchServer struct {
	grpc.ServerStream
}

func (x *watchServiceCreateWatchServer) Send(m *Event) error {
	return x.ServerStream.SendMsg(m)
}

func _WatchService_SubscribeToTopic_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(Topic)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(WatchServiceServer).SubscribeToTopic(m, &watchServiceSubscribeToTopicServer{stream})
}

type WatchService_SubscribeToTopicServer interface {
	Send(*Event) error
	grpc.ServerStream
}

type watchServiceSubscribeToTopicServer struct {
	grpc.ServerStream
}

func (x *watchServiceSubscribeToTopicServer) Send(m *Event) error {
	return x.ServerStream.SendMsg(m)
}

// WatchService_ServiceDesc is the grpc.ServiceDesc for WatchService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var WatchService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "apis.v1.WatchService",
	HandlerType: (*WatchServiceServer)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "CreateWatch",
			Handler:       _WatchService_CreateWatch_Handler,
			ServerStreams: true,
		},
		{
			StreamName:    "SubscribeToTopic",
			Handler:       _WatchService_SubscribeToTopic_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "watch.proto",
}
