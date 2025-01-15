package s3

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
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
		WithMainBucket("test"),
		WithS3Config(func(config *aws.Config) *aws.Config {
			config = config.WithEndpoint(endpointURL)
			config = config.WithS3ForcePathStyle(true)
			config = config.WithRegion("r")
			return config
		}),
		WithS3Credentionals(accessKey, secretKey))
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
