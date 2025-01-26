package data

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtensionByContentType(t *testing.T) {
	tests := []struct {
		contentType string
		ext         any
	}{
		{
			contentType: `text/plain`,
			ext:         `.txt`,
		},
		{
			contentType: `text/html`,
			ext:         `.html`,
		},
		{
			contentType: `application/xml`,
			ext:         `.xml`,
		},
		{
			contentType: `application/json`,
			ext:         `.json`,
		},
		{
			contentType: `application/javascript`,
			ext:         `.js`,
		},
		{
			contentType: `image/jpeg`,
			ext:         []string{`.jpg`, `.jpeg`, `.jfif`, `.pjpeg`, `.pjp`},
		},
		{
			contentType: `undefined`,
			ext:         ``,
		},
	}
	for _, test := range tests {
		assert.Contains(t, test.ext, ExtensionByContentType(test.contentType))
	}
}
