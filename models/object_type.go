//
// @project apfs 2018
// @author Dmitry Ponomarev <demdxx@gmail.com> 2018
//

package models

import (
	"strings"
)

// ObjectType of object
type ObjectType string

// Object types...
const (
	TypeUndefined    ObjectType = "undefined"
	TypeVideo        ObjectType = "video"
	TypeImage        ObjectType = "image"
	TypeAudio        ObjectType = "audio"
	TypeHTMLArchType ObjectType = "htmlarch"
	TypeOther        ObjectType = "other"
)

// ObjectTypeByContentType value
func ObjectTypeByContentType(contentType string) ObjectType {
	switch {
	case contentType == "image" || strings.HasPrefix(contentType, "image/"):
		return TypeImage
	case contentType == "video" || strings.HasPrefix(contentType, "video/"):
		return TypeVideo
	case contentType == "audio" || strings.HasPrefix(contentType, "audio/"):
		return TypeAudio
	case contentType == "htmlarch":
		return TypeHTMLArchType
	}
	return TypeOther
}

// ObjectTypesByContentType value list
func ObjectTypesByContentType(contentTypes ...string) (list []ObjectType) {
	for _, contentType := range contentTypes {
		if contentType == "*" {
			return nil // Allowed everething
		}
		list = append(list, ObjectTypeByContentType(contentType))
	}
	return
}

func (t ObjectType) String() string {
	return string(t)
}

// IsUndefined type object
func (t ObjectType) IsUndefined() bool {
	return t == TypeUndefined || t == ""
}

// IsVideo file type
func (t ObjectType) IsVideo() bool {
	return t == TypeVideo
}

// IsImage file type
func (t ObjectType) IsImage() bool {
	return t == TypeImage
}
