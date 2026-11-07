package relay

import "context"

type mockServer struct {
	listenCalls   int
	listenErr     error
	shutdownCalls int
	shutdownErr   error

	called chan struct{}
}

func newMockServer() *mockServer {
	return &mockServer{
		called: make(chan struct{}),
	}
}

func (m *mockServer) ListenAndServe() error {
	m.listenCalls++

	select {
	case <-m.called:
	default:
		close(m.called)
	}

	return m.listenErr
}

func (m *mockServer) Shutdown(context.Context) error {
	m.shutdownCalls++
	return m.shutdownErr
}
