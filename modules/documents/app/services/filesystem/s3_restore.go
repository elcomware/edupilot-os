package filesystem

import (
	"context"
	"crypto/sha256"
	"errors"
	"slices"
	"strings"

	"github.com/elcomware/edupilot/modules/documents/app/services"
	"github.com/elcomware/edupilot/platform/tenantupgrade"
	"github.com/minio/minio-go/v7"
)

var errBucketEvidence = errors.New("restored bucket failed verification")

// ReconcileS3Restore inspects current objects in a dedicated frozen recovery bucket.
// The operator supplies read-only credentials separately from the application's.
func ReconcileS3Restore(ctx context.Context, get func(string, string) string, manifest tenantupgrade.Manifest, refs tenantupgrade.ObjectReferences, expected string) (ReconciledLocalInventory, error) {
	if manifest.Validate() != nil || refs.Validate() != nil || len(expected) != 64 || !strings.HasPrefix(get("S3_ENDPOINT", ""), "https://") {
		return ReconciledLocalInventory{}, errBucketEvidence
	}
	schools := make([]string, len(manifest.Schools))
	for i, m := range manifest.Schools {
		schools[i] = m.LegacySchool
	}
	slices.Sort(schools)
	if !slices.Equal(schools, refs.Schools) {
		return ReconciledLocalInventory{}, errBucketEvidence
	}
	store, err := configuredS3(get)
	if err != nil {
		return ReconciledLocalInventory{}, errBucketEvidence
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	listed := map[string]minio.ObjectInfo{}
	wanted := make(map[string]bool, len(refs.Objects))
	for _, r := range refs.Objects {
		wanted["schools/"+r.School+"/documents/"+r.ID] = true
	}
	for item := range store.client.ListObjectsIter(ctx, store.bucket, minio.ListObjectsOptions{Recursive: true}) {
		if item.Err != nil || !wanted[item.Key] || item.ETag == "" || item.Size <= 0 || item.Size > int64(services.MaxRecordSize) {
			return ReconciledLocalInventory{}, errBucketEvidence
		}
		delete(wanted, item.Key)
		listed[item.Key] = item
	}
	if len(wanted) != 0 || ctx.Err() != nil {
		return ReconciledLocalInventory{}, errBucketEvidence
	}
	scan := localInspection{result: LocalInventory{Version: LocalInventoryVersion, Schools: len(schools)}, digest: sha256.New()}
	scan.beginSchools(schools)
	for _, r := range refs.Objects {
		item := listed["schools/"+r.School+"/documents/"+r.ID]
		opts := minio.GetObjectOptions{}
		if err := opts.SetMatchETag(item.ETag); err != nil {
			return ReconciledLocalInventory{}, errBucketEvidence
		}
		object, err := store.client.GetObject(ctx, store.bucket, item.Key, opts)
		if err != nil {
			return ReconciledLocalInventory{}, errBucketEvidence
		}
		k, doc, err := services.VerifyStoredRecord(r.School, r.ID, object)
		closeErr := object.Close()
		recordSize := int64(len(doc.Content) + (services.MaxRecordSize - services.MaxFileSize))
		if err != nil || closeErr != nil || recordSize != item.Size {
			return ReconciledLocalInventory{}, errBucketEvidence
		}
		if err := scan.addDocument(k, doc); err != nil {
			return ReconciledLocalInventory{}, errBucketEvidence
		}
	}
	inventory, err := scan.finish(expected)
	if err != nil || ctx.Err() != nil {
		return ReconciledLocalInventory{}, errBucketEvidence
	}
	return reconciledInventory(inventory, refs)
}
