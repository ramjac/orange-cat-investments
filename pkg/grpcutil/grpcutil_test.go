package grpcutil

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/orange-cat-investments/oci/pkg/mtls"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type echoRequest struct {
	Message string `json:"message"`
}

type echoResponse struct {
	Message string `json:"message"`
}

type echoService interface {
	Echo(ctx context.Context, in *echoRequest) (*echoResponse, error)
	EchoStream(stream grpc.ServerStream) error
}

type echoServiceImpl struct{}

func (s *echoServiceImpl) Echo(ctx context.Context, in *echoRequest) (*echoResponse, error) {
	return &echoResponse{Message: "echo:" + in.Message}, nil
}

func (s *echoServiceImpl) EchoStream(stream grpc.ServerStream) error {
	req := new(echoRequest)
	if err := stream.RecvMsg(req); err != nil {
		return err
	}
	return stream.SendMsg(&echoResponse{Message: "stream:" + req.Message})
}

func _Echo_Handler(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	in := new(echoRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(echoService).Echo(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/EchoService/Echo",
	}
	handler := func(ctx context.Context, req any) (any, error) {
		return srv.(echoService).Echo(ctx, req.(*echoRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _EchoStream_Handler(srv any, stream grpc.ServerStream) error {
	return srv.(echoService).EchoStream(stream)
}

var echoServiceDesc = grpc.ServiceDesc{
	ServiceName: "EchoService",
	HandlerType: (*echoService)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Echo",
			Handler:    _Echo_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "EchoStream",
			Handler:       _EchoStream_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
	},
}

func TestCleanTarget(t *testing.T) {
	assert.Equal(t, "localhost:50051", CleanTarget("http://localhost:50051"))
	assert.Equal(t, "workforce-api-service:50051", CleanTarget("https://workforce-api-service:50051/"))
	assert.Equal(t, "127.0.0.1:8080", CleanTarget("127.0.0.1:8080"))
}

func TestGRPCUtil_RoundTrip(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer lis.Close()

	srv := NewServer(nil)
	srv.RegisterService(&echoServiceDesc, &echoServiceImpl{})
	go func() {
		_ = srv.Serve(lis)
	}()
	defer srv.Stop()

	conn, err := Dial(context.Background(), "http://"+lis.Addr().String(), nil)
	require.NoError(t, err)
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	resp := new(echoResponse)
	err = conn.Invoke(ctx, "/EchoService/Echo", &echoRequest{Message: "test"}, resp)
	require.NoError(t, err)
	assert.Equal(t, "echo:test", resp.Message)
}

type testPKI struct {
	dir    string
	caFile string
	caCert *x509.Certificate
	caKey  *ecdsa.PrivateKey
}

func newTestPKI(t *testing.T) *testPKI {
	t.Helper()
	dir := t.TempDir()

	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	require.NoError(t, err)

	caTmpl := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: "oci-test-ca"},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	caDER, err := x509.CreateCertificate(rand.Reader, caTmpl, caTmpl, &caKey.PublicKey, caKey)
	require.NoError(t, err)
	caCert, err := x509.ParseCertificate(caDER)
	require.NoError(t, err)

	caFile := filepath.Join(dir, "ca.crt")
	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})
	require.NoError(t, os.WriteFile(caFile, caPEM, 0600))

	return &testPKI{
		dir:    dir,
		caFile: caFile,
		caCert: caCert,
		caKey:  caKey,
	}
}

func (p *testPKI) issueCert(t *testing.T, name string, dnsNames []string, ips []net.IP, spiffeURIs []string) (certFile, keyFile string) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	require.NoError(t, err)

	var parsedURIs []*url.URL
	for _, raw := range spiffeURIs {
		u, parseErr := url.Parse(raw)
		require.NoError(t, parseErr)
		parsedURIs = append(parsedURIs, u)
	}

	tmpl := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: name},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		DNSNames:              dnsNames,
		IPAddresses:           ips,
		URIs:                  parsedURIs,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, p.caCert, &key.PublicKey, p.caKey)
	require.NoError(t, err)
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)

	certFile = filepath.Join(p.dir, name+".crt")
	keyFile = filepath.Join(p.dir, name+".key")
	require.NoError(t, os.WriteFile(certFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER}), 0600))
	require.NoError(t, os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}), 0600))
	return certFile, keyFile
}

func invokeStreamEcho(ctx context.Context, conn *grpc.ClientConn, msg string) (*echoResponse, error) {
	desc := &grpc.StreamDesc{
		StreamName:    "EchoStream",
		ServerStreams: true,
		ClientStreams: true,
	}
	stream, err := conn.NewStream(ctx, desc, "/EchoService/EchoStream")
	if err != nil {
		return nil, err
	}
	if err := stream.SendMsg(&echoRequest{Message: msg}); err != nil {
		return nil, err
	}
	if err := stream.CloseSend(); err != nil {
		return nil, err
	}
	resp := new(echoResponse)
	if err := stream.RecvMsg(resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func startMTLSServer(t *testing.T, pki *testPKI, srvCert, srvKey string, allowedSPIFFEIDs ...string) string {
	t.Helper()
	serverTLS, err := mtls.NewServerTLSConfig(srvCert, srvKey, pki.caFile)
	require.NoError(t, err)

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = lis.Close() })

	srv := NewServer(serverTLS, WithAllowedPeerSPIFFEIDs(allowedSPIFFEIDs...)...)
	srv.RegisterService(&echoServiceDesc, &echoServiceImpl{})
	go func() {
		_ = srv.Serve(lis)
	}()
	t.Cleanup(srv.Stop)
	return lis.Addr().String()
}

func TestResolveNamespaceAndAllowedCallers(t *testing.T) {
	t.Setenv("POD_NAMESPACE", "")
	t.Setenv("ALLOWED_PEER_SPIFFE_IDS", "")
	assert.Equal(t, "oci-core", ResolveNamespace())

	t.Setenv("POD_NAMESPACE", "  oci-dev \n")
	assert.Equal(t, "oci-dev", ResolveNamespace())

	assert.Equal(t, []string{
		"spiffe://oci.local/ns/oci-core/sa/customer-bll",
		"spiffe://oci.local/ns/oci-core/sa/internal-bll",
		"spiffe://oci.local/ns/oci-core/sa/oci-worker",
	}, AllowedDomainAPICallers("oci-core"))

	assert.Equal(t, []string{
		"spiffe://oci.local/ns/oci-core/sa/oci-customer-bff",
		"spiffe://oci.local/ns/oci-core/sa/oci-worker",
	}, AllowedCustomerBLLCallers("oci-core"))

	assert.Equal(t, []string{
		"spiffe://oci.local/ns/oci-core/sa/oci-employee-bff",
		"spiffe://oci.local/ns/oci-core/sa/oci-pebble-proxy",
		"spiffe://oci.local/ns/oci-core/sa/oci-worker",
		"spiffe://oci.local/ns/external-sim/sa/mock-hardware",
	}, AllowedInternalBLLCallers("oci-core"))

	assert.Equal(t, []string{
		"spiffe://oci.local/ns/oci-dev/sa/oci-employee-bff",
		"spiffe://oci.local/ns/oci-dev/sa/oci-pebble-proxy",
		"spiffe://oci.local/ns/oci-dev/sa/oci-worker",
		"spiffe://oci.local/ns/external-sim/sa/mock-hardware-dev",
	}, AllowedInternalBLLCallers("oci-dev"))

	t.Setenv("ALLOWED_PEER_SPIFFE_IDS", " spiffe://oci.local/ns/custom/sa/a , spiffe://oci.local/ns/custom/sa/b ")
	expectedOverride := []string{
		"spiffe://oci.local/ns/custom/sa/a",
		"spiffe://oci.local/ns/custom/sa/b",
	}
	assert.Equal(t, expectedOverride, AllowedDomainAPICallers("oci-core"))
	assert.Equal(t, expectedOverride, AllowedCustomerBLLCallers("oci-core"))
	assert.Equal(t, expectedOverride, AllowedInternalBLLCallers("oci-core"))
}

func TestSPIFFEAuthInterceptor_DomainAPI(t *testing.T) {
	t.Setenv("ALLOWED_PEER_SPIFFE_IDS", "")
	pki := newTestPKI(t)
	srvCert, srvKey := pki.issueCert(
		t,
		"workforce-api",
		[]string{"localhost", "workforce-api-service.oci-core.svc.cluster.local"},
		[]net.IP{net.ParseIP("127.0.0.1")},
		[]string{"spiffe://oci.local/ns/oci-core/sa/workforce-api"},
	)

	addr := startMTLSServer(t, pki, srvCert, srvKey, AllowedDomainAPICallers("oci-core")...)

	tests := []struct {
		name      string
		spiffeIDs []string
		wantCode  codes.Code
	}{
		{
			name:      "customer-bll allowed",
			spiffeIDs: []string{"spiffe://oci.local/ns/oci-core/sa/customer-bll"},
			wantCode:  codes.OK,
		},
		{
			name:      "internal-bll allowed",
			spiffeIDs: []string{"spiffe://oci.local/ns/oci-core/sa/internal-bll"},
			wantCode:  codes.OK,
		},
		{
			name:      "oci-worker allowed",
			spiffeIDs: []string{"spiffe://oci.local/ns/oci-core/sa/oci-worker"},
			wantCode:  codes.OK,
		},
		{
			name:      "oci-customer-bff denied on Domain API",
			spiffeIDs: []string{"spiffe://oci.local/ns/oci-core/sa/oci-customer-bff"},
			wantCode:  codes.PermissionDenied,
		},
		{
			name:      "lateral workforce-api denied on Domain API",
			spiffeIDs: []string{"spiffe://oci.local/ns/oci-core/sa/workforce-api"},
			wantCode:  codes.PermissionDenied,
		},
		{
			name:      "wrong namespace denied",
			spiffeIDs: []string{"spiffe://oci.local/ns/oci-dev/sa/customer-bll"},
			wantCode:  codes.PermissionDenied,
		},
		{
			name:      "certificate without URI SAN denied",
			spiffeIDs: nil,
			wantCode:  codes.PermissionDenied,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			certFile, keyFile := pki.issueCert(t, tc.name, nil, nil, tc.spiffeIDs)
			clientTLS, err := mtls.NewClientTLSConfig(certFile, keyFile, pki.caFile)
			require.NoError(t, err)

			conn, err := Dial(context.Background(), addr, clientTLS)
			require.NoError(t, err)
			defer conn.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			resp := new(echoResponse)
			err = conn.Invoke(ctx, "/EchoService/Echo", &echoRequest{Message: "hello"}, resp)
			if tc.wantCode == codes.OK {
				require.NoError(t, err)
				assert.Equal(t, "echo:hello", resp.Message)
			} else {
				require.Error(t, err)
				assert.Equal(t, tc.wantCode, status.Code(err))
			}

			streamResp, streamErr := invokeStreamEcho(ctx, conn, "hello")
			if tc.wantCode == codes.OK {
				require.NoError(t, streamErr)
				assert.Equal(t, "stream:hello", streamResp.Message)
			} else {
				require.Error(t, streamErr)
				assert.Equal(t, tc.wantCode, status.Code(streamErr))
			}
		})
	}
}

func TestSPIFFEAuthInterceptor_BLLCallers(t *testing.T) {
	t.Setenv("ALLOWED_PEER_SPIFFE_IDS", "")
	pki := newTestPKI(t)

	custBLLCert, custBLLKey := pki.issueCert(
		t, "customer-bll",
		[]string{"localhost"}, []net.IP{net.ParseIP("127.0.0.1")},
		[]string{"spiffe://oci.local/ns/oci-core/sa/customer-bll"},
	)
	custBLLAddr := startMTLSServer(t, pki, custBLLCert, custBLLKey, AllowedCustomerBLLCallers("oci-core")...)

	intBLLCert, intBLLKey := pki.issueCert(
		t, "internal-bll",
		[]string{"localhost"}, []net.IP{net.ParseIP("127.0.0.1")},
		[]string{"spiffe://oci.local/ns/oci-core/sa/internal-bll"},
	)
	intBLLAddr := startMTLSServer(t, pki, intBLLCert, intBLLKey, AllowedInternalBLLCallers("oci-core")...)

	custBFFCert, custBFFKey := pki.issueCert(t, "cust-bff", nil, nil, []string{"spiffe://oci.local/ns/oci-core/sa/oci-customer-bff"})
	empBFFCert, empBFFKey := pki.issueCert(t, "emp-bff", nil, nil, []string{"spiffe://oci.local/ns/oci-core/sa/oci-employee-bff"})
	pebbleCert, pebbleKey := pki.issueCert(t, "pebble", nil, nil, []string{"spiffe://oci.local/ns/oci-core/sa/oci-pebble-proxy"})
	mockHWCert, mockHWKey := pki.issueCert(t, "mock-hw", nil, nil, []string{"spiffe://oci.local/ns/external-sim/sa/mock-hardware"})

	callUnary := func(t *testing.T, addr, certFile, keyFile string) error {
		t.Helper()
		clientTLS, err := mtls.NewClientTLSConfig(certFile, keyFile, pki.caFile)
		require.NoError(t, err)
		conn, err := Dial(context.Background(), addr, clientTLS)
		require.NoError(t, err)
		defer conn.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		return conn.Invoke(ctx, "/EchoService/Echo", &echoRequest{Message: "ping"}, new(echoResponse))
	}

	require.NoError(t, callUnary(t, custBLLAddr, custBFFCert, custBFFKey))
	assert.Equal(t, codes.PermissionDenied, status.Code(callUnary(t, custBLLAddr, empBFFCert, empBFFKey)))

	require.NoError(t, callUnary(t, intBLLAddr, empBFFCert, empBFFKey))
	require.NoError(t, callUnary(t, intBLLAddr, pebbleCert, pebbleKey))
	require.NoError(t, callUnary(t, intBLLAddr, mockHWCert, mockHWKey))
	assert.Equal(t, codes.PermissionDenied, status.Code(callUnary(t, intBLLAddr, custBFFCert, custBFFKey)))
}

type mockServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (m *mockServerStream) Context() context.Context {
	return m.ctx
}

func TestSPIFFEAuthInterceptors_MissingPeerContext(t *testing.T) {
	_, err := ExtractPeerSPIFFEIDs(context.Background())
	require.Error(t, err)

	unary := UnarySPIFFEAuthInterceptor("spiffe://oci.local/ns/oci-core/sa/customer-bll")
	_, err = unary(context.Background(), &echoRequest{}, &grpc.UnaryServerInfo{FullMethod: "/EchoService/Echo"}, func(ctx context.Context, req any) (any, error) {
		return &echoResponse{}, nil
	})
	require.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))

	stream := StreamSPIFFEAuthInterceptor("spiffe://oci.local/ns/oci-core/sa/customer-bll")
	err = stream(nil, &mockServerStream{ctx: context.Background()}, &grpc.StreamServerInfo{FullMethod: "/EchoService/EchoStream"}, func(srv any, ss grpc.ServerStream) error {
		return nil
	})
	require.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestPureSPIFFESVIDServerCertVerification(t *testing.T) {
	pki := newTestPKI(t)

	// Pure SPIFFE SVID server certificate: URI SAN only, zero DNSNames and zero IPAddresses.
	srvCert, srvKey := pki.issueCert(
		t,
		"pure-spiffe-workforce-api",
		nil,
		nil,
		[]string{"spiffe://oci.local/ns/oci-core/sa/workforce-api"},
	)
	addr := startMTLSServer(t, pki, srvCert, srvKey, "spiffe://oci.local/ns/oci-core/sa/customer-bll")

	clientCert, clientKey := pki.issueCert(
		t,
		"customer-bll-client",
		nil,
		nil,
		[]string{"spiffe://oci.local/ns/oci-core/sa/customer-bll"},
	)

	t.Run("matching service authority succeeds", func(t *testing.T) {
		clientTLS, err := mtls.NewClientTLSConfig(clientCert, clientKey, pki.caFile)
		require.NoError(t, err)

		conn, err := Dial(
			context.Background(),
			addr,
			clientTLS,
			grpc.WithAuthority("workforce-api-service.oci-core.svc.cluster.local:50051"),
		)
		require.NoError(t, err)
		defer conn.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		resp := new(echoResponse)
		err = conn.Invoke(ctx, "/EchoService/Echo", &echoRequest{Message: "spiffe"}, resp)
		require.NoError(t, err)
		assert.Equal(t, "echo:spiffe", resp.Message)
	})

	t.Run("wrong service authority fails", func(t *testing.T) {
		clientTLS, err := mtls.NewClientTLSConfig(clientCert, clientKey, pki.caFile)
		require.NoError(t, err)

		conn, err := Dial(
			context.Background(),
			addr,
			clientTLS,
			grpc.WithAuthority("ops-api-service.oci-core.svc.cluster.local:50054"),
		)
		require.NoError(t, err)
		defer conn.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		err = conn.Invoke(ctx, "/EchoService/Echo", &echoRequest{Message: "spiffe"}, new(echoResponse))
		require.Error(t, err)
	})

	t.Run("wrong namespace authority fails", func(t *testing.T) {
		clientTLS, err := mtls.NewClientTLSConfig(clientCert, clientKey, pki.caFile)
		require.NoError(t, err)

		conn, err := Dial(
			context.Background(),
			addr,
			clientTLS,
			grpc.WithAuthority("workforce-api-service.oci-dev.svc.cluster.local:50051"),
		)
		require.NoError(t, err)
		defer conn.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		err = conn.Invoke(ctx, "/EchoService/Echo", &echoRequest{Message: "spiffe"}, new(echoResponse))
		require.Error(t, err)
	})
}
