package backends

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/ansible/receptor/pkg/framer"
	"github.com/ansible/receptor/pkg/logger"
	"github.com/ansible/receptor/pkg/netceptor"
)

func TestNewTCPDialer(t *testing.T) {
	type args struct {
		address string
		redial  bool
		tls     *tls.Config
		logger  *logger.ReceptorLogger
	}
	tests := []struct {
		name    string
		args    args
		want    *TCPDialer
		wantErr bool
	}{
		{
			name: "Positive",
			args: args{
				address: "127.0.0.1:9999",
				redial:  true,
				tls:     nil,
				logger:  nil,
			},
			want: &TCPDialer{
				address: "127.0.0.1:9999",
				redial:  true,
				tls:     nil,
				logger:  nil,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewTCPDialer(tt.args.address, tt.args.redial, tt.args.tls, tt.args.logger)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewTCPDialer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewTCPDialer() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestTCPDialerGetAddr(t *testing.T) {
	tests := []struct {
		name    string
		address string
		want    string
	}{
		{
			name:    "Basic",
			address: "127.0.0.1:9999",
			want:    "127.0.0.1:9999",
		},
		{
			name:    "Empty",
			address: "",
			want:    "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &TCPDialer{
				address: tt.address,
			}
			if got := b.GetAddr(); got != tt.want {
				t.Errorf("TCPDialer.GetAddr() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTCPDialerGetTLS(t *testing.T) {
	tlsConfig := &tls.Config{}
	tests := []struct {
		name string
		tls  *tls.Config
		want *tls.Config
	}{
		{
			name: "With TLS",
			tls:  tlsConfig,
			want: tlsConfig,
		},
		{
			name: "Without TLS",
			tls:  nil,
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &TCPDialer{
				tls: tt.tls,
			}
			if got := b.GetTLS(); got != tt.want {
				t.Errorf("TCPDialer.GetTLS() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTCPListenerGetAddr(t *testing.T) {
	addr := &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 9999}

	tests := []struct {
		name string
		li   net.Listener
		want string
	}{
		{
			name: "Basic",
			li: &mockNetListener{
				addr: addr,
			},
			want: "127.0.0.1:9999",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &TCPListener{
				li: tt.li,
			}
			if got := b.GetAddr(); got != tt.want {
				t.Errorf("TCPListener.GetAddr() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTCPListenerGetCost(t *testing.T) {
	addr := &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 9999}

	tests := []struct {
		name string
		li   net.Listener
		want string
	}{
		{
			name: "Basic",
			li: &mockNetListener{
				addr: addr,
			},
			want: "127.0.0.1:9999",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &TCPListener{
				li: tt.li,
			}
			if got := b.GetCost(); got != tt.want {
				t.Errorf("TCPListener.GetCost() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTCPListenerGetTLS(t *testing.T) {
	tlsConfig := &tls.Config{}
	tests := []struct {
		name string
		TLS  *tls.Config
		want *tls.Config
	}{
		{
			name: "With TLS",
			TLS:  tlsConfig,
			want: tlsConfig,
		},
		{
			name: "Without TLS",
			TLS:  nil,
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &TCPListener{
				TLS: tt.TLS,
			}
			if got := b.GetTLS(); got != tt.want {
				t.Errorf("TCPListener.GetTLS() = %v, want %v", got, tt.want)
			}
		})
	}
}

// mockNetListener is a mock implementation of net.Listener for testing
type mockNetListener struct {
	addr net.Addr
}

func (m *mockNetListener) Accept() (net.Conn, error) {
	return nil, nil
}

func (m *mockNetListener) Close() error {
	return nil
}

func (m *mockNetListener) Addr() net.Addr {
	return m.addr
}

func TestNewTCPListener(t *testing.T) {
	type args struct {
		address string
		tls     *tls.Config
		logger  *logger.ReceptorLogger
	}
	tests := []struct {
		name    string
		args    args
		want    *TCPListener
		wantErr bool
	}{
		{
			name: "Positive",
			args: args{
				address: "127.0.0.1:9999",
				tls:     nil,
				logger:  nil,
			},
			want: &TCPListener{
				address: "127.0.0.1:9999",
				TLS:     nil,
				li:      nil,
				innerLi: nil,
				logger:  nil,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewTCPListener(tt.args.address, tt.args.tls, tt.args.logger)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewTCPListener() error = %v, wantErr %v", err, tt.wantErr)

				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewTCPListener() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestTCPListenerStart(t *testing.T) {
	type fields struct {
		address string
		TLS     *tls.Config
		li      net.Listener
		innerLi *net.TCPListener
		logger  *logger.ReceptorLogger
	}
	type args struct {
		ctx context.Context
		wg  *sync.WaitGroup
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    chan netceptor.BackendSession
		wantErr bool
	}{
		{
			name: "Positive",
			fields: fields{
				address: "127.0.0.1:9998",
				TLS:     nil,
				li:      nil,
				innerLi: nil,
				logger:  logger.NewReceptorLogger("TCPtest"),
			},
			args: args{
				ctx: context.Background(),
				wg:  &sync.WaitGroup{},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &TCPListener{
				address: tt.fields.address,
				TLS:     tt.fields.TLS,
				li:      tt.fields.li,
				innerLi: tt.fields.innerLi,
				logger:  tt.fields.logger,
			}
			got, err := b.Start(tt.args.ctx, tt.args.wg)
			if (err != nil) != tt.wantErr {
				t.Errorf("TCPListener.Start() error = %+v, wantErr %+v", err, tt.wantErr)

				return
			}
			if got == nil {
				t.Errorf("TCPListener.Start() returned nil")
			}
		})
	}
}

func TestTCPDialerStart(t *testing.T) {
	type fields struct {
		address string
		redial  bool
		tls     *tls.Config
		logger  *logger.ReceptorLogger
	}
	type args struct {
		ctx context.Context
		wg  *sync.WaitGroup
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    chan netceptor.BackendSession
		wantErr bool
	}{
		{
			name: "Positive",
			fields: fields{
				address: "127.0.0.1:9998",
				redial:  true,
				tls:     nil,
				logger:  logger.NewReceptorLogger("TCPtest"),
			},
			args: args{
				ctx: context.Background(),
				wg:  &sync.WaitGroup{},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &TCPDialer{
				address: tt.fields.address,
				redial:  tt.fields.redial,
				tls:     tt.fields.tls,
				logger:  tt.fields.logger,
			}
			got, err := b.Start(tt.args.ctx, tt.args.wg)
			if (err != nil) != tt.wantErr {
				t.Errorf("TCPDialer.Start() error = %+v, wantErr %+v", err, tt.wantErr)

				return
			}
			if got == nil {
				t.Errorf("TCPDialer.Start() got = nil")
			}
		})
	}
}

// mockConn is a mock implementation of net.Conn for testing
type mockConn struct {
	readData  []byte
	readIndex int
	writeData []byte
	closed    bool
	readErr   error
	writeErr  error
}

func newMockConn(readData []byte) *mockConn {
	return &mockConn{
		readData:  readData,
		readIndex: 0,
		writeData: []byte{},
		closed:    false,
	}
}

func (m *mockConn) Read(b []byte) (n int, err error) {
	if m.readErr != nil {
		return 0, m.readErr
	}
	if m.closed {
		return 0, io.EOF
	}
	if m.readIndex >= len(m.readData) {
		return 0, io.EOF
	}
	n = copy(b, m.readData[m.readIndex:])
	m.readIndex += n
	return n, nil
}

func (m *mockConn) Write(b []byte) (n int, err error) {
	if m.writeErr != nil {
		return 0, m.writeErr
	}
	if m.closed {
		return 0, fmt.Errorf("connection closed")
	}
	m.writeData = append(m.writeData, b...)
	return len(b), nil
}

func (m *mockConn) Close() error {
	m.closed = true
	return nil
}

func (m *mockConn) LocalAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 12345}
}

func (m *mockConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 54321}
}

func (m *mockConn) SetDeadline(t time.Time) error {
	return nil
}

func (m *mockConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (m *mockConn) SetWriteDeadline(t time.Time) error {
	return nil
}

func TestNewTCPSession(t *testing.T) {
	conn := newMockConn([]byte{})
	closeChan := make(chan struct{})

	session := newTCPSession(conn, closeChan)

	if session == nil {
		t.Errorf("newTCPSession() returned nil")
	}

	if session.conn != conn {
		t.Errorf("newTCPSession() did not set conn correctly")
	}

	if session.closeChan != closeChan {
		t.Errorf("newTCPSession() did not set closeChan correctly")
	}

	if session.framer == nil {
		t.Errorf("newTCPSession() did not initialize framer")
	}
}

func TestTCPSessionSend(t *testing.T) {
	tests := []struct {
		name      string
		data      []byte
		writeErr  error
		wantErr   bool
		wantWrite []byte
	}{
		{
			name:      "Success",
			data:      []byte("test data"),
			writeErr:  nil,
			wantErr:   false,
			wantWrite: framer.New().SendData([]byte("test data")),
		},
		{
			name:      "Write error",
			data:      []byte("test data"),
			writeErr:  fmt.Errorf("write error"),
			wantErr:   true,
			wantWrite: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := newMockConn([]byte{})
			conn.writeErr = tt.writeErr

			session := newTCPSession(conn, nil)

			err := session.Send(tt.data)

			if (err != nil) != tt.wantErr {
				t.Errorf("TCPSession.Send() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantWrite != nil && !reflect.DeepEqual(conn.writeData, tt.wantWrite) {
				t.Errorf("TCPSession.Send() wrote %v, want %v", conn.writeData, tt.wantWrite)
			}
		})
	}
}

func TestTCPSessionRecv(t *testing.T) {
	// Create a framed message for testing
	testData := []byte("test data")
	framedData := framer.New().SendData(testData)

	tests := []struct {
		name     string
		readData []byte
		readErr  error
		timeout  time.Duration
		wantData []byte
		wantErr  bool
	}{
		{
			name:     "Success",
			readData: framedData,
			readErr:  nil,
			timeout:  time.Second,
			wantData: testData,
			wantErr:  false,
		},
		{
			name:     "Read error",
			readData: framedData,
			readErr:  fmt.Errorf("read error"),
			timeout:  time.Second,
			wantData: nil,
			wantErr:  true,
		},
		{
			name:     "Timeout",
			readData: []byte{}, // Empty data will cause timeout
			readErr:  nil,
			timeout:  1 * time.Millisecond,
			wantData: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := newMockConn(tt.readData)
			conn.readErr = tt.readErr

			session := newTCPSession(conn, nil)

			gotData, err := session.Recv(tt.timeout)

			if (err != nil) != tt.wantErr {
				t.Errorf("TCPSession.Recv() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantData != nil && !reflect.DeepEqual(gotData, tt.wantData) {
				t.Errorf("TCPSession.Recv() = %v, want %v", gotData, tt.wantData)
			}
		})
	}
}

func TestTCPSessionClose(t *testing.T) {
	tests := []struct {
		name      string
		closeChan chan struct{}
	}{
		{
			name:      "With close channel",
			closeChan: make(chan struct{}),
		},
		{
			name:      "Without close channel",
			closeChan: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := newMockConn([]byte{})

			session := newTCPSession(conn, tt.closeChan)

			err := session.Close()

			if err != nil {
				t.Errorf("TCPSession.Close() error = %v", err)
			}

			if !conn.closed {
				t.Errorf("TCPSession.Close() did not close the connection")
			}

			if tt.closeChan != nil {
				// Verify the close channel is closed
				select {
				case _, ok := <-tt.closeChan:
					if ok {
						t.Errorf("TCPSession.Close() did not close the channel")
					}
				default:
					t.Errorf("TCPSession.Close() did not close the channel")
				}
			}
		})
	}
}

func TestTCPListenerCfgGetCost(t *testing.T) {
	tests := []struct {
		name string
		cost float64
		want float64
	}{
		{
			name: "Positive cost",
			cost: 2.5,
			want: 2.5,
		},
		{
			name: "Zero cost",
			cost: 0.0,
			want: 0.0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := TCPListenerCfg{
				Cost: tt.cost,
			}
			if got := cfg.GetCost(); got != tt.want {
				t.Errorf("TCPListenerCfg.GetCost() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTCPListenerCfgGetNodeCost(t *testing.T) {
	tests := []struct {
		name     string
		nodeCost map[string]float64
		want     map[string]float64
	}{
		{
			name:     "With node costs",
			nodeCost: map[string]float64{"node1": 1.5, "node2": 2.5},
			want:     map[string]float64{"node1": 1.5, "node2": 2.5},
		},
		{
			name:     "Empty node costs",
			nodeCost: map[string]float64{},
			want:     map[string]float64{},
		},
		{
			name:     "Nil node costs",
			nodeCost: nil,
			want:     nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := TCPListenerCfg{
				NodeCost: tt.nodeCost,
			}
			got := cfg.GetNodeCost()
			if (got == nil) != (tt.want == nil) {
				t.Errorf("TCPListenerCfg.GetNodeCost() = %v, want %v", got, tt.want)
				return
			}
			if got != nil {
				if len(got) != len(tt.want) {
					t.Errorf("TCPListenerCfg.GetNodeCost() = %v, want %v", got, tt.want)
					return
				}
				for k, v := range got {
					if tt.want[k] != v {
						t.Errorf("TCPListenerCfg.GetNodeCost()[%s] = %v, want %v", k, v, tt.want[k])
					}
				}
			}
		})
	}
}

func TestTCPListenerCfgGetAddr(t *testing.T) {
	tests := []struct {
		name     string
		bindAddr string
		want     string
	}{
		{
			name:     "With bind address",
			bindAddr: "127.0.0.1",
			want:     "127.0.0.1",
		},
		{
			name:     "Empty bind address",
			bindAddr: "",
			want:     "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := TCPListenerCfg{
				BindAddr: tt.bindAddr,
			}
			if got := cfg.GetAddr(); got != tt.want {
				t.Errorf("TCPListenerCfg.GetAddr() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTCPListenerCfgGetTLS(t *testing.T) {
	tests := []struct {
		name string
		tls  string
		want string
	}{
		{
			name: "With TLS",
			tls:  "tls-config",
			want: "tls-config",
		},
		{
			name: "Empty TLS",
			tls:  "",
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := TCPListenerCfg{
				TLS: tt.tls,
			}
			if got := cfg.GetTLS(); got != tt.want {
				t.Errorf("TCPListenerCfg.GetTLS() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTCPListenerCfgPrepare(t *testing.T) {
	tests := []struct {
		name     string
		cost     float64
		nodeCost map[string]float64
		wantErr  bool
	}{
		{
			name:     "Valid cost",
			cost:     1.0,
			nodeCost: map[string]float64{"node1": 1.5},
			wantErr:  false,
		},
		{
			name:     "Zero cost",
			cost:     0.0,
			nodeCost: map[string]float64{},
			wantErr:  true,
		},
		{
			name:     "Negative cost",
			cost:     -1.0,
			nodeCost: map[string]float64{},
			wantErr:  true,
		},
		{
			name:     "Negative node cost",
			cost:     1.0,
			nodeCost: map[string]float64{"node1": -1.0},
			wantErr:  true,
		},
		{
			name:     "Zero node cost",
			cost:     1.0,
			nodeCost: map[string]float64{"node1": 0.0},
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := TCPListenerCfg{
				Cost:     tt.cost,
				NodeCost: tt.nodeCost,
			}
			err := cfg.Prepare()
			if (err != nil) != tt.wantErr {
				t.Errorf("TCPListenerCfg.Prepare() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTCPDialerCfgPrepare(t *testing.T) {
	tests := []struct {
		name    string
		cost    float64
		wantErr bool
	}{
		{
			name:    "Valid cost",
			cost:    1.0,
			wantErr: false,
		},
		{
			name:    "Zero cost",
			cost:    0.0,
			wantErr: true,
		},
		{
			name:    "Negative cost",
			cost:    -1.0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := TCPDialerCfg{
				Cost: tt.cost,
			}
			err := cfg.Prepare()
			if (err != nil) != tt.wantErr {
				t.Errorf("TCPDialerCfg.Prepare() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTCPDialerCfgPreReload(t *testing.T) {
	tests := []struct {
		name    string
		cost    float64
		wantErr bool
	}{
		{
			name:    "Valid cost",
			cost:    1.0,
			wantErr: false,
		},
		{
			name:    "Zero cost",
			cost:    0.0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := TCPDialerCfg{
				Cost: tt.cost,
			}
			err := cfg.PreReload()
			if (err != nil) != tt.wantErr {
				t.Errorf("TCPDialerCfg.PreReload() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTCPListenerCfgPreReload(t *testing.T) {
	tests := []struct {
		name    string
		cost    float64
		wantErr bool
	}{
		{
			name:    "Valid cost",
			cost:    1.0,
			wantErr: false,
		},
		{
			name:    "Zero cost",
			cost:    0.0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := TCPListenerCfg{
				Cost: tt.cost,
			}
			err := cfg.PreReload()
			if (err != nil) != tt.wantErr {
				t.Errorf("TCPListenerCfg.PreReload() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Skip tests for Run and Reload methods since they depend on the global netceptor.MainInstance
// which is difficult to mock properly. These methods are simple wrappers around
// NewTCPListener/NewTCPDialer and AddBackend, which are tested elsewhere.
func TestTCPListenerCfgRunSkip(t *testing.T) {
	t.Skip("Skipping test for TCPListenerCfg.Run() as it depends on global netceptor.MainInstance")
}

func TestTCPDialerCfgRunSkip(t *testing.T) {
	t.Skip("Skipping test for TCPDialerCfg.Run() as it depends on global netceptor.MainInstance")
}

func TestTCPListenerCfgReloadSkip(t *testing.T) {
	t.Skip("Skipping test for TCPListenerCfg.Reload() as it depends on global netceptor.MainInstance")
}

func TestTCPDialerCfgReloadSkip(t *testing.T) {
	t.Skip("Skipping test for TCPDialerCfg.Reload() as it depends on global netceptor.MainInstance")
}
