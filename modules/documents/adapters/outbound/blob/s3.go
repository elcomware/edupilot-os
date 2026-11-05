package blob

import (
	"bytes"
	"context"
	"errors"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/elcomware/edupilot-os/modules/documents/application"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/s3utils"
)

type s3Store struct {
	client *minio.Client
	bucket string
}

// Configured builds a provider without touching the filesystem or making network requests.
func Configured(get func(string, string) string) (application.BlobStore, error) {
	switch get("EDUPILOT_DOCUMENTS_DRIVER", "local") {
	case "local":
		root := get("EDUPILOT_DOCUMENTS_ROOT", "storage/documents")
		if strings.TrimSpace(root) == "" {
			return nil, errors.New("missing document root")
		}
		abs, err := filepath.Abs(root)
		if err != nil {
			return nil, err
		}
		return local{abs}, nil
	case "s3":
		endpoint, err := url.Parse(get("S3_ENDPOINT", ""))
		if err != nil || endpoint.Hostname() == "" || (endpoint.Scheme != "http" && endpoint.Scheme != "https") || endpoint.User != nil || endpoint.RawQuery != "" || endpoint.Fragment != "" || (endpoint.Path != "" && endpoint.Path != "/") {
			return nil, errors.New("invalid S3 endpoint")
		}
		bucket, access, secret := get("S3_BUCKET", ""), get("S3_ACCESS_KEY_ID", ""), get("S3_SECRET_ACCESS_KEY", "")
		if s3utils.CheckValidBucketNameStrict(bucket) != nil || access == "" || secret == "" {
			return nil, errors.New("invalid S3 bucket or missing credentials")
		}
		client, err := minio.New(endpoint.Host, &minio.Options{Creds: credentials.NewStaticV4(access, secret, ""), Secure: endpoint.Scheme == "https", Region: get("S3_REGION", "us-east-1")})
		if err != nil {
			return nil, err
		}
		return s3Store{client, bucket}, nil
	default:
		return nil, errors.New("unknown document storage driver")
	}
}

func (s s3Store) Put(ctx context.Context, k application.FileKey, record []byte) error {
	opts := minio.PutObjectOptions{ContentType: "application/octet-stream", DisableMultipart: true}
	opts.SetMatchETagExcept("*")
	_, err := s.client.PutObject(ctx, s.bucket, k.StoragePath(), bytes.NewReader(record), int64(len(record)), opts)
	return err
}

func (s s3Store) Get(ctx context.Context, k application.FileKey) ([]byte, error) {
	object, err := s.client.GetObject(ctx, s.bucket, k.StoragePath(), minio.GetObjectOptions{})
	if err != nil {
		return nil, s3ReadError(err)
	}
	defer object.Close()
	record, err := application.ReadRecord(object)
	if err != nil {
		return nil, s3ReadError(err)
	}
	return record, nil
}

func s3ReadError(err error) error {
	switch minio.ToErrorResponse(err).Code {
	case "NoSuchKey", "NotFound", "AccessDenied":
		// Without ListBucket, S3 may deliberately respond with AccessDenied for a missing key.
		return application.ErrFileMissing
	default:
		return err
	}
}
