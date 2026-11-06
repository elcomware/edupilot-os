package config

import (
	"fmt"
	"net"
	"strings"

	"github.com/elcomware/edupilot/app/facades"
	"google.golang.org/grpc/credentials"
)

// Goravel applies transport credentials before it registers any RPC services.
func GrpcServerCredentials() credentials.TransportCredentials {
	c := facades.Config()
	creds, err := grpcCredentials(c.GetString("grpc.host"), c.GetString("grpc.cert"), c.GetString("grpc.key"))
	if err != nil {
		panic(err)
	}
	return creds
}

func grpcCredentials(host, cert, key string) (credentials.TransportCredentials, error) {
	if host == "" {
		return nil, nil
	}
	if cert != "" || key != "" {
		if cert == "" || key == "" {
			return nil, fmt.Errorf("GRPC_TLS_CERT and GRPC_TLS_KEY must be supplied together")
		}
		return credentials.NewServerTLSFromFile(cert, key)
	}
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	if ip := net.ParseIP(strings.Trim(host, "[]")); ip != nil && ip.IsLoopback() {
		return nil, nil
	}
	return nil, fmt.Errorf("a non-loopback gRPC listener requires GRPC_TLS_CERT and GRPC_TLS_KEY")
}
