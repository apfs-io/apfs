package validation

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/apfs-io/apfs/models"
)

func TestMaxSizeValidator(t *testing.T) {
	v := MaxSizeValidator(1024)

	err := v.Validate(context.Background(), &ValidationRequest{Size: 512})
	assert.NoError(t, err)

	err = v.Validate(context.Background(), &ValidationRequest{Size: 2048})
	require.Error(t, err)
	assert.True(t, IsValidationError(err))
}

func TestMinSizeValidator(t *testing.T) {
	v := MinSizeValidator(100)

	err := v.Validate(context.Background(), &ValidationRequest{Size: 200})
	assert.NoError(t, err)

	err = v.Validate(context.Background(), &ValidationRequest{Size: 50})
	require.Error(t, err)
	assert.True(t, IsValidationError(err))
}

func TestContentTypeValidator(t *testing.T) {
	v := ContentTypeValidator([]string{"video/*", "image/jpeg"})

	err := v.Validate(context.Background(), &ValidationRequest{ContentType: "video/mp4"})
	assert.NoError(t, err)

	err = v.Validate(context.Background(), &ValidationRequest{ContentType: "image/jpeg"})
	assert.NoError(t, err)

	err = v.Validate(context.Background(), &ValidationRequest{ContentType: "text/plain"})
	require.Error(t, err)
	assert.True(t, IsValidationError(err))
}

func TestContentTypeValidatorSniff(t *testing.T) {
	jpegHeader := "\xff\xd8\xff\xe0"
	v := ContentTypeValidator([]string{"image/*"})
	err := v.Validate(context.Background(), &ValidationRequest{
		Reader: strings.NewReader(jpegHeader + "rest-of-file"),
	})
	assert.NoError(t, err)
}

func TestChain(t *testing.T) {
	v := Chain(
		MaxSizeValidator(1024),
		MinSizeValidator(10),
	)

	err := v.Validate(context.Background(), &ValidationRequest{Size: 500})
	assert.NoError(t, err)

	err = v.Validate(context.Background(), &ValidationRequest{Size: 2000})
	require.Error(t, err)
}

func TestFromWorkflowValidate(t *testing.T) {
	wv := &models.WorkflowValidate{
		MaxSize:      "1MB",
		ContentTypes: []string{"image/*"},
	}
	v := FromWorkflowValidate(wv, nil)

	err := v.Validate(context.Background(), &ValidationRequest{
		Size:        512 * 1024,
		ContentType: "image/png",
	})
	assert.NoError(t, err)

	err = v.Validate(context.Background(), &ValidationRequest{
		Size:        2 * 1024 * 1024,
		ContentType: "image/png",
	})
	require.Error(t, err) // too large
}
