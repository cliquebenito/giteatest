package gitaly

import (
	"context"
	"encoding/base64"
	"sync"

	"code.gitea.io/gitea/modules/json"
	"code.gitea.io/gitea/modules/setting"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	gitalyauth "gitlab.com/gitlab-org/gitaly/v16/auth"
	gitalyclient "gitlab.com/gitlab-org/gitaly/v16/client"
	"gitlab.com/gitlab-org/gitaly/v16/proto/go/gitalypb"
)

type cacheKey struct {
	address, token string
}

func getCacheKey() cacheKey {
	return cacheKey{address: setting.Gitaly.GitalyServers[setting.Gitaly.MainServerName].Address, token: setting.Gitaly.GitalyServers[setting.Gitaly.MainServerName].Token}
}

type connectionsCache struct {
	sync.RWMutex
	connections map[cacheKey]*grpc.ClientConn
}

var (
	// This connection cache map contains two types of connections:
	// - Normal gRPC connections
	// - Sidechannel connections. When client dials to the Gitaly server, the
	// server multiplexes the connection using Yamux. In the future, the server
	// can open another stream to transfer data without gRPC. Besides, we apply
	// a framing protocol to add the half-close capability to Yamux streams.
	// Hence, we cannot use those connections interchangeably.
	cache = connectionsCache{
		connections: make(map[cacheKey]*grpc.ClientConn),
	}
	sidechannelRegistry *gitalyclient.SidechannelRegistry

	connectionsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sourcecontrol_gitaly_connections_total",
			Help: "Number of Gitaly connections that have been established",
		},
		[]string{"status"},
	)
)

func InitializeSidechannelRegistry() {
	if sidechannelRegistry == nil {
		accessLogger := log.New()
		accessLogger.SetLevel(log.InfoLevel)
		sidechannelRegistry = gitalyclient.NewSidechannelRegistry(logrus.NewEntry(accessLogger))
	}
}

var allowedMetadataKeys = map[string]bool{
	"user_id":   true,
	"username":  true,
	"remote_ip": true,
}

func withOutgoingMetadata(ctx context.Context) context.Context {
	md := metadata.New(nil)

	marshaledServers, err := json.Marshal(setting.Gitaly.GitalyServers)
	if err != nil {
		log.Errorf("Error has occurred while marshal gitaly servers: %v", err)
		return nil
	}

	md.Set("gitaly-servers", base64.StdEncoding.EncodeToString(marshaledServers))
	return metadata.NewOutgoingContext(ctx, md)
}

func NewSmartHTTPClient(ctx context.Context) (context.Context, *SmartHTTPClient, error) {
	conn, err := getOrCreateConnection()
	if err != nil {
		return nil, nil, err
	}
	grpcClient := gitalypb.NewSmartHTTPServiceClient(conn)
	smartHTTPClient := &SmartHTTPClient{
		SmartHTTPServiceClient: grpcClient,
		sidechannelRegistry:    sidechannelRegistry,
	}
	return withOutgoingMetadata(ctx), smartHTTPClient, nil
}

func NewBlobClient(ctx context.Context) (context.Context, *BlobClient, error) {
	conn, err := getOrCreateConnection()
	if err != nil {
		return nil, nil, err
	}
	grpcClient := gitalypb.NewBlobServiceClient(conn)
	return withOutgoingMetadata(ctx), &BlobClient{grpcClient}, nil
}

func NewRepositoryClient(ctx context.Context) (context.Context, *RepositoryClient, error) {
	conn, err := getOrCreateConnection()
	if err != nil {
		return nil, nil, err
	}
	grpcClient := gitalypb.NewRepositoryServiceClient(conn)
	return withOutgoingMetadata(ctx), &RepositoryClient{grpcClient}, nil
}

func NewDiffClient(ctx context.Context) (context.Context, *DiffClient, error) {
	conn, err := getOrCreateConnection()
	if err != nil {
		return nil, nil, err
	}
	grpcClient := gitalypb.NewDiffServiceClient(conn)
	return withOutgoingMetadata(ctx), &DiffClient{grpcClient}, nil
}

func NewRefClient(ctx context.Context) (context.Context, *RefClient, error) {
	conn, err := getOrCreateConnection()
	if err != nil {
		return nil, nil, err
	}
	grpcClient := gitalypb.NewRefServiceClient(conn)
	return withOutgoingMetadata(ctx), &RefClient{grpcClient}, nil
}

func NewCommitClient(ctx context.Context) (context.Context, *CommitClient, error) {
	conn, err := getOrCreateConnection()
	if err != nil {
		return nil, nil, err
	}
	grpcClient := gitalypb.NewCommitServiceClient(conn)
	return withOutgoingMetadata(ctx), &CommitClient{grpcClient}, nil
}

func NewOperationClient(ctx context.Context) (context.Context, *OperationClient, error) {
	conn, err := getOrCreateConnection()
	if err != nil {
		return nil, nil, err
	}
	grpcClient := gitalypb.NewOperationServiceClient(conn)
	return withOutgoingMetadata(ctx), &OperationClient{grpcClient}, nil
}

func NewSSHClient(ctx context.Context) (context.Context, *SSHServiceClient, error) {
	conn, err := getOrCreateConnection()
	if err != nil {
		return nil, nil, err
	}
	grpcClient := gitalypb.NewSSHServiceClient(conn)
	return withOutgoingMetadata(ctx), &SSHServiceClient{
		SSHServiceClient: grpcClient,
	}, nil
}

func NewConflictsClient(ctx context.Context) (context.Context, *ConflictsClient, error) {
	conn, err := getOrCreateConnection()
	if err != nil {
		return nil, nil, err
	}
	grpcClient := gitalypb.NewConflictsServiceClient(conn)
	return withOutgoingMetadata(ctx), &ConflictsClient{grpcClient}, nil
}

func getOrCreateConnection() (*grpc.ClientConn, error) {
	key := getCacheKey()

	cache.RLock()
	conn := cache.connections[key]
	cache.RUnlock()

	if conn != nil {
		return conn, nil
	}

	cache.Lock()
	defer cache.Unlock()

	if conn := cache.connections[key]; conn != nil {
		return conn, nil
	}

	conn, err := newConnection()
	if err != nil {
		return nil, err
	}

	cache.connections[key] = conn

	return conn, nil
}

func CloseConnections() {
	cache.Lock()
	defer cache.Unlock()

	for _, conn := range cache.connections {
		conn.Close()
	}
}

func newConnection() (*grpc.ClientConn, error) {
	connOpts := append(
		gitalyclient.DefaultDialOpts,
		grpc.WithPerRPCCredentials(gitalyauth.RPCCredentialsV2(setting.Gitaly.GitalyServers[setting.Gitaly.MainServerName].Token)),
		grpc.WithChainStreamInterceptor(grpc_prometheus.StreamClientInterceptor),
		grpc.WithChainUnaryInterceptor(grpc_prometheus.UnaryClientInterceptor),
		// In https://gitlab.com/groups/gitlab-org/-/epics/8971, we added DNS discovery support to Praefect. This was
		// done by making two changes:
		// - Configure client-side round-robin load-balancing in client dial options. We added that as a default option
		// inside gitaly client in gitaly client since v15.9.0
		// - Configure DNS resolving. Due to some technical limitations, we don't use gRPC's built-in DNS resolver.
		// Instead, we implement our own DNS resolver. This resolver is exposed via the following configuration.
		// Afterward, workhorse can detect and handle DNS discovery automatically. The user needs to setup and set
		// Gitaly address to something like "dns:gitaly.service.dc1.consul"
		gitalyclient.WithGitalyDNSResolver(gitalyclient.DefaultDNSResolverBuilderConfig()),
	)

	conn, connErr := gitalyclient.DialSidechannel(context.Background(), setting.Gitaly.GitalyServers[setting.Gitaly.MainServerName].Address, sidechannelRegistry, connOpts) // lint:allow context.Background

	label := "ok"
	if connErr != nil {
		label = "fail"
	}
	connectionsTotal.WithLabelValues(label).Inc()

	return conn, connErr
}

func UnmarshalJSON(s string, msg proto.Message) error {
	return protojson.UnmarshalOptions{DiscardUnknown: true}.Unmarshal([]byte(s), msg)
}
