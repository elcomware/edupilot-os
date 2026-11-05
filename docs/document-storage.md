# Private document storage

The Documents module stores uploads for one school and returns an opaque UUID. A bearer session supplies the school. Admins and registrars can upload and download. Viewers can download a file when they have its UUID. This is a school-wide file permission model. Use a narrower access policy before storing records limited to a particular student, family or staff group.

| Request | Permission | Result |
| --- | --- | --- |
| `POST /api/v1/documents/files` | `documents.write` | Send the raw bytes and a matching `Content-Type`. A 201 response contains `{"id":"…"}` and a `Location` header. |
| `GET /api/v1/documents/files/{id}` | `documents.read` | Returns the original bytes as a private download, with a filename derived from the UUID and media type. |

Supply `Authorization: Bearer <token>` to both requests. Uploads must contain at least one byte and may be up to 8 MiB. The server accepts PDF, PNG, JPEG and UTF-8 plain text. It checks the declared type against the file signature and validates the full encoding of plain text. The checks identify supported formats. They do not scan for malware or sanitize PDFs. A malformed, unknown or foreign UUID returns 404. The client must keep the UUID when it needs to retrieve the file later.

With a session token in `TOKEN` and an API origin in `API`, upload and download a text file as follows.

```sh
curl --fail-with-body -H "Authorization: Bearer ${TOKEN}" \
  -H 'Content-Type: text/plain' --data-binary @notice.txt \
  "${API}/api/v1/documents/files"

# Set DOCUMENT_ID to the id in the response.
curl --fail-with-body -H "Authorization: Bearer ${TOKEN}" \
  "${API}/api/v1/documents/files/${DOCUMENT_ID}" -o downloaded-notice.txt
```

## Choose a provider

| Setting | Local | S3-compatible storage |
| --- | --- | --- |
| `EDUPILOT_DOCUMENTS_DRIVER` | `local` (default) | `s3` |
| `EDUPILOT_DOCUMENTS_ROOT` | Defaults to `storage/documents`, relative to the process working directory. | Unused |
| `S3_ENDPOINT` | Unused | Origin with scheme and host, such as `http://storage:9000` inside the development Compose network. Use HTTPS for remote deployments. |
| `S3_BUCKET` | Unused | Name of a private, pre-created bucket. |
| `S3_ACCESS_KEY_ID`, `S3_SECRET_ACCESS_KEY` | Unused | Credentials for a scoped service account. |
| `S3_REGION` | Unused | Defaults to `us-east-1`. Set the actual bucket region for regional services. |

The local provider needs a private, persistent directory writable by the API process. Keep it outside any public or static file root. New directories have mode 0700 and new records have mode 0600 on POSIX systems. The filesystem must support hard links. An upload is published after its temporary file has been written and synced.

For S3, create the bucket and a scoped access key in the MinIO console or your provider's administration tool. Keep the bucket private. Grant the runtime account `s3:GetObject` and `s3:PutObject` on `arn:aws:s3:::YOUR_BUCKET/schools/*/documents/*`. The service uses conditional writes and does not list buckets or create them. A denied object read is reported to the client as missing. Configure server-side encryption and access logs in the storage provider according to the school's retention policy.

The storage provider initializes on first file use. Changes to backend selection, root, endpoint or credentials take effect after restarting the API. To move existing files, copy all stored records with their original keys before switching the provider. Switching alone does not transfer data.

## Check integrity and recovery

A record is stored at `schools/{verified-school}/documents/{uuid}`. Its versioned envelope contains a SHA-256 checksum over the storage key and original bytes. Before sending any bytes, the API verifies the checksum, maximum size and file type. A damaged record, or a record copied to another school's key, yields a generic storage error. The checksum detects corruption and misplaced records. Control write access to the storage account and retain independent backups, because a writer with direct backend access could replace both bytes and checksum.

Back up the whole private directory or S3 prefix, including each record envelope, and retain the document IDs wherever a workflow references them. Restore records under the original keys. The school work-list endpoint keeps its usual behavior.

The Go suite includes local round-trip, tenant isolation, role, format, oversize, symlink and corruption cases. To exercise S3, provision a disposable private bucket and export `EDUPILOT_TEST_S3_ENDPOINT`, `EDUPILOT_TEST_S3_BUCKET`, `EDUPILOT_TEST_S3_ACCESS_KEY_ID`, `EDUPILOT_TEST_S3_SECRET_ACCESS_KEY` and optionally `EDUPILOT_TEST_S3_REGION`. The opt-in test uploads and removes one generated file in a unique integration-school prefix. Its test account needs `s3:DeleteObject` for cleanup.
