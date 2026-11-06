package config

import "testing"

func TestGrpcCredentialsRequiredOutsideLoopback(t *testing.T) {
	for _, host := range []string{"", "127.0.0.1", "[::1]:50051"} {
		if _, err := grpcCredentials(host, "", ""); err != nil {
			t.Errorf("%s: %v", host, err)
		}
	}
	for _, tc := range [][3]string{{"0.0.0.0", "", ""}, {"localhost", "", ""}, {":50051", "", ""}, {"192.0.2.1", "cert", ""}, {"127.0.0.1", "", "key"}, {"127.0.0.1", "missing", "missing"}} {
		if _, err := grpcCredentials(tc[0], tc[1], tc[2]); err == nil {
			t.Error("unsafe or invalid configuration accepted")
		}
	}
}
