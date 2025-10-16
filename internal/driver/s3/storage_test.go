package s3

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

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
		WithS3Credentionals(accessKey, secretKey),
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
