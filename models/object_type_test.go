package models

import (
	"testing"
)

func TestObjectTypeByContentType(t *testing.T) {
	tests := []struct {
		contentType string
		targetType  ObjectType
	}{
		{
			contentType: "image/png",
			targetType:  TypeImage,
		},
		{
			contentType: "video/mpeg",
			targetType:  TypeVideo,
		},
		{
			contentType: "audio/mp3",
			targetType:  TypeAudio,
		},
		{
			contentType: "htmlarch",
			targetType:  TypeHTMLArchType,
		},
		{
			contentType: "text/html",
			targetType:  TypeOther,
		},
	}
	for _, test := range tests {
		if ObjectTypeByContentType(test.contentType) != test.targetType {
			t.Errorf("invalid type detection: %s", test.targetType)
		}
	}
}
