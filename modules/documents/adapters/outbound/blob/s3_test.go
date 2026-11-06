package blob

import (
	"bytes"
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/elcomware/edupilot/modules/documents/application"
	"github.com/elcomware/edupilot/platform/identity"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
)

func TestS3RejectsIncompleteConfiguration(t *testing.T) {
	for _, endpoint := range []string{"ftp://example.test", "https://user:pass@example.test", "https://example.test/uploads", "http://localhost:9000"} {
		_, err := Configured(func(key, fallback string) string {
			switch key {
			case "EDUPILOT_DOCUMENTS_DRIVER":
				return "s3"
			case "S3_ENDPOINT":
				return endpoint
			default:
				return fallback
			}
		})
		if err == nil {
			t.Fatalf("accepted invalid/incomplete configuration for %q", endpoint)
		}
	}
}

// Set EDUPILOT_TEST_S3_ENDPOINT and the other EDUPILOT_TEST_S3_* variables to
// enable this round-trip on a private, disposable integration bucket.
func TestS3PrivateBucketRoundTrip(t *testing.T) {
	if os.Getenv("EDUPILOT_TEST_S3_ENDPOINT") == "" {
		t.Skip("dedicated S3 integration bucket is not configured")
	}
	store, err := Configured(func(key, fallback string) string {
		if key == "EDUPILOT_DOCUMENTS_DRIVER" {
			return "s3"
		}
		if v := os.Getenv("EDUPILOT_TEST_" + key); v != "" {
			return v
		}
		return fallback
	})
	if err != nil {
		t.Fatal(err)
	}
	fixtureID, err := uuid.NewRandom()
	if err != nil {
		t.Fatal(err)
	}
	school := "integration-" + fixtureID.String()
	files := application.Files{Resolve: func() (application.BlobStore, error) { return store, nil }}
	content := []byte("round-trip document")
	authorized := identity.WithPrincipal(t.Context(), identity.Principal{ID: "fixture", School: school, Permissions: []string{identity.DocumentsRead, identity.DocumentsWrite}})
	id, err := files.Upload(authorized, "text/plain", content)
	if err != nil {
		t.Fatal(err)
	}
	s := store.(s3Store)
	t.Cleanup(func() {
		ctx, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		if err := s.client.RemoveObject(ctx, s.bucket, "schools/"+school+"/documents/"+id, minio.RemoveObjectOptions{}); err != nil {
			t.Errorf("remove test document: %v", err)
		}
	})
	got, err := files.Download(authorized, id)
	if err != nil || !bytes.Equal(content, got.Content) {
		t.Fatalf("S3 download failed: %v", err)
	}
	if _, err := files.Download(identity.WithPrincipal(t.Context(), identity.Principal{ID: "other", School: "other-school", Permissions: []string{identity.DocumentsRead}}), id); !errors.Is(err, application.ErrFileMissing) {
		t.Fatalf("foreign-school lookup: %v", err)
	}
}
