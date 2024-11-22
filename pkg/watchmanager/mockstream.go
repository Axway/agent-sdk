package watchmanager

import (
	"context"

	"google.golang.org/grpc/metadata"

	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
	"google.golang.org/grpc"
)

type mockWatchClient struct {
	stream *mockStream
	err    error
}

func (m mockWatchClient) Subscribe(ctx context.Context, _ ...grpc.CallOption) (proto.Watch_SubscribeClient, error) {
	m.stream.context = ctx
	return m.stream, m.err
}

type mockConn struct {
	stream *mockStream
}

func (m mockConn) Invoke(_ context.Context, _ string, _ interface{}, _ interface{}, _ ...grpc.CallOption) error {
	return nil
}

func (m mockConn) NewStream(
	ctx context.Context,
	_ *grpc.StreamDesc,
	_ string,
	_ ...grpc.CallOption,
) (grpc.ClientStream, error) {
	m.stream.context = ctx
	return m.stream, nil
}

type mockStream struct {
	event   *proto.Event
	err     error
	context context.Context
	request *proto.Request
}

func (m *mockStream) Send(request *proto.Request) error {
	if m.err != nil {
		return m.err
	}

	m.request = request
	return nil
}

func (m *mockStream) Recv() (*proto.Event, error) {
	return m.event, m.err
}

func (m *mockStream) Header() (metadata.MD, error) {
	return metadata.MD{}, nil
}

func (m *mockStream) Trailer() metadata.MD {
	return metadata.MD{}
}

func (m *mockStream) CloseSend() error {
	return nil
}

func (m *mockStream) Context() context.Context {
	return m.context
}

func (m *mockStream) SendMsg(_ interface{}) error {
	return nil
}

func (m *mockStream) RecvMsg(_ interface{}) error {
	return nil
}

func newMockWatchClient(stream *mockStream, err error) newWatchClientFunc {
	return func(_ grpc.ClientConnInterface) proto.WatchClient {
		return &mockWatchClient{
			stream: stream,
			err:    err,
		}
	}
}
