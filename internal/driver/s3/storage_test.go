package s3

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/apfs-io/apfs/models"
)

func TestS3Collection(t *testing.T) {
	endpointURL := os.Getenv(`TEST_S3_ENDPOINTURL`)
	accessKey := os.Getenv("TEST_S3_ACCESS_KEY")
	secretKey := os.Getenv("TEST_S3_SECRET_KEY")
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second*10)
	defer cancel()

	if endpointURL == `` {
		t.SkipNow()
		return
	}

	collection, err := NewStorage(
		ctx,
		WithMainBucket("test"),
		WithRegion("r"),
		WithEndpoint(endpointURL),
		WithS3Credentials(accessKey, secretKey),
	)
	assert.NoError(t, err, `new collection`)

	object, err := collection.Create(ctx, "assets", nil, false, nil)
	assert.NoError(t, err, "create `asserts` bucket")

	err = collection.Update(ctx, object, models.OriginalFilename,
		bytes.NewReader([]byte(`data`)), &models.ItemMeta{ContentType: "text"})
	assert.NoError(t, err, "create new file object")

	dataStream, err := collection.Read(ctx, object, models.OriginalFilename)
	assert.NoError(t, err, "get original object data")

	data, err := io.ReadAll(dataStream)
	assert.NoError(t, err, "read data content")

	assert.Equal(t, []byte(`data`), data, "data matching")
}

// TestCreateMissingGroupManifest is the upload-404 regression: Create must
// treat a missing bucket-level manifest.json as empty workflow, not return
// the leftover GetObject NoSuchKey to the caller.
func TestCreateMissingGroupManifest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(fakeS3NoManifest))
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection, err := NewStorage(
		ctx,
		WithMainBucket("assets"),
		WithRegion("us-east-1"),
		WithEndpoint(srv.URL),
		WithInsecure(true),
		WithS3Credentials("test", "testtest"),
	)
	require.NoError(t, err)

	object, err := collection.Create(ctx, "image", nil, false, nil)
	require.NoError(t, err, "Create must ignore missing group manifest.json")
	require.NotNil(t, object)
	assert.Equal(t, "image", object.Bucket())
	assert.NotEmpty(t, object.Path())
}

func fakeS3NoManifest(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(r.URL.Path, "/")
	w.Header().Set("Content-Type", "application/xml")

	switch r.Method {
	case http.MethodGet, http.MethodHead:
		if path == "" {
			_, _ = io.WriteString(w, `<?xml version="1.0" encoding="UTF-8"?>
<ListAllMyBucketsResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Owner><ID>minio</ID><DisplayName>minio</DisplayName></Owner>
  <Buckets><Bucket><Name>assets</Name><CreationDate>2020-01-01T00:00:00.000Z</CreationDate></Bucket></Buckets>
</ListAllMyBucketsResult>`)
			return
		}
		bucket, key, _ := strings.Cut(path, "/")
		_ = bucket
		if key == "" {
			_, _ = io.WriteString(w, `<?xml version="1.0" encoding="UTF-8"?>
<ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Name>assets</Name><IsTruncated>false</IsTruncated>
</ListBucketResult>`)
			return
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `<?xml version="1.0" encoding="UTF-8"?>
<Error><Code>NoSuchKey</Code><Message>The specified key does not exist.</Message><Key>`+key+`</Key></Error>`)
	case http.MethodPut:
		w.WriteHeader(http.StatusOK)
	default:
		w.WriteHeader(http.StatusOK)
	}
}
