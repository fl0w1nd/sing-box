package daemon

import (
	context "context"

	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	ExtendedService_GetCapabilities_FullMethodName            = "/daemon.ExtendedService/GetCapabilities"
	ExtendedService_SubscribeOutboundProviders_FullMethodName = "/daemon.ExtendedService/SubscribeOutboundProviders"
	ExtendedService_UpdateOutboundProvider_FullMethodName     = "/daemon.ExtendedService/UpdateOutboundProvider"
)

// ExtendedServiceClient is the client API for ExtendedService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ExtendedServiceClient interface {
	GetCapabilities(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*ExtensionCapabilities, error)
	SubscribeOutboundProviders(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (grpc.ServerStreamingClient[OutboundProviderStatusUpdate], error)
	UpdateOutboundProvider(ctx context.Context, in *OutboundProviderRequest, opts ...grpc.CallOption) (*emptypb.Empty, error)
}

type extendedServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewExtendedServiceClient(cc grpc.ClientConnInterface) ExtendedServiceClient {
	return &extendedServiceClient{cc}
}

func (c *extendedServiceClient) GetCapabilities(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*ExtensionCapabilities, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(ExtensionCapabilities)
	err := c.cc.Invoke(ctx, ExtendedService_GetCapabilities_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *extendedServiceClient) SubscribeOutboundProviders(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (grpc.ServerStreamingClient[OutboundProviderStatusUpdate], error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	stream, err := c.cc.NewStream(ctx, &ExtendedService_ServiceDesc.Streams[0], ExtendedService_SubscribeOutboundProviders_FullMethodName, cOpts...)
	if err != nil {
		return nil, err
	}
	x := &grpc.GenericClientStream[emptypb.Empty, OutboundProviderStatusUpdate]{ClientStream: stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type ExtendedService_SubscribeOutboundProvidersClient = grpc.ServerStreamingClient[OutboundProviderStatusUpdate]

func (c *extendedServiceClient) UpdateOutboundProvider(ctx context.Context, in *OutboundProviderRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, ExtendedService_UpdateOutboundProvider_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ExtendedServiceServer is the server API for ExtendedService service.
// All implementations must embed UnimplementedExtendedServiceServer
// for forward compatibility.
type ExtendedServiceServer interface {
	GetCapabilities(context.Context, *emptypb.Empty) (*ExtensionCapabilities, error)
	SubscribeOutboundProviders(*emptypb.Empty, grpc.ServerStreamingServer[OutboundProviderStatusUpdate]) error
	UpdateOutboundProvider(context.Context, *OutboundProviderRequest) (*emptypb.Empty, error)
	mustEmbedUnimplementedExtendedServiceServer()
}

// UnimplementedExtendedServiceServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedExtendedServiceServer struct{}

func (UnimplementedExtendedServiceServer) GetCapabilities(context.Context, *emptypb.Empty) (*ExtensionCapabilities, error) {
	return nil, status.Error(codes.Unimplemented, "method GetCapabilities not implemented")
}

func (UnimplementedExtendedServiceServer) SubscribeOutboundProviders(*emptypb.Empty, grpc.ServerStreamingServer[OutboundProviderStatusUpdate]) error {
	return status.Error(codes.Unimplemented, "method SubscribeOutboundProviders not implemented")
}

func (UnimplementedExtendedServiceServer) UpdateOutboundProvider(context.Context, *OutboundProviderRequest) (*emptypb.Empty, error) {
	return nil, status.Error(codes.Unimplemented, "method UpdateOutboundProvider not implemented")
}
func (UnimplementedExtendedServiceServer) mustEmbedUnimplementedExtendedServiceServer() {}
func (UnimplementedExtendedServiceServer) testEmbeddedByValue()                         {}

// UnsafeExtendedServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ExtendedServiceServer will
// result in compilation errors.
type UnsafeExtendedServiceServer interface {
	mustEmbedUnimplementedExtendedServiceServer()
}

func RegisterExtendedServiceServer(s grpc.ServiceRegistrar, srv ExtendedServiceServer) {
	// If the following call panics, it indicates UnimplementedExtendedServiceServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&ExtendedService_ServiceDesc, srv)
}

func _ExtendedService_GetCapabilities_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ExtendedServiceServer).GetCapabilities(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ExtendedService_GetCapabilities_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ExtendedServiceServer).GetCapabilities(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _ExtendedService_SubscribeOutboundProviders_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(emptypb.Empty)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(ExtendedServiceServer).SubscribeOutboundProviders(m, &grpc.GenericServerStream[emptypb.Empty, OutboundProviderStatusUpdate]{ServerStream: stream})
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type ExtendedService_SubscribeOutboundProvidersServer = grpc.ServerStreamingServer[OutboundProviderStatusUpdate]

func _ExtendedService_UpdateOutboundProvider_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(OutboundProviderRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ExtendedServiceServer).UpdateOutboundProvider(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ExtendedService_UpdateOutboundProvider_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ExtendedServiceServer).UpdateOutboundProvider(ctx, req.(*OutboundProviderRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// ExtendedService_ServiceDesc is the grpc.ServiceDesc for ExtendedService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var ExtendedService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "daemon.ExtendedService",
	HandlerType: (*ExtendedServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetCapabilities",
			Handler:    _ExtendedService_GetCapabilities_Handler,
		},
		{
			MethodName: "UpdateOutboundProvider",
			Handler:    _ExtendedService_UpdateOutboundProvider_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "SubscribeOutboundProviders",
			Handler:       _ExtendedService_SubscribeOutboundProviders_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "daemon/extended_service.proto",
}
