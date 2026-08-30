package client

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyGroupPrefix(t *testing.T) {
	assert.Equal(t, "image/2026/08/8/abc", applyGroupPrefix("image/2026/08/8/abc", "default"),
		"fully-qualified IDs must not get default/ prefixed")
	assert.Equal(t, "image/2026/08/8/abc", applyGroupPrefix("image/2026/08/8/abc", "image"))
	assert.Equal(t, "image/abc", applyGroupPrefix("abc", "image"))
	assert.Equal(t, "abc", applyGroupPrefix("abc", ""))
	assert.Equal(t, "", applyGroupPrefix("", "default"))
}

func TestToProtoObjectID_FullyQualified(t *testing.T) {
	id := &ObjectID{Id: "image/2026/08/8/84d1aaffdd6822fed447725e9cb7d8bc", Name: []string{"orig.jpg"}}
	got := toProtoObjectID(id, "default")
	require.NotNil(t, got)
	assert.Equal(t, id.Id, got.Id)
	assert.Equal(t, []string{"orig.jpg"}, got.Name)
}

func TestToProtoObjectID_BareName(t *testing.T) {
	got := toProtoObjectID(&ObjectID{Id: "abc"}, "image")
	assert.Equal(t, "image/abc", got.Id)
}

func TestToProtoObjectIDNames_FullyQualified(t *testing.T) {
	got := toProtoObjectIDNames(&ObjectIDNames{Id: "image/obj", Names: []string{"thumb.jpg"}}, "default")
	assert.Equal(t, "image/obj", got.Id)
}

func TestPrepareObjectID_FullyQualified(t *testing.T) {
	id := &ObjectID{Id: "image/obj"}
	assert.Equal(t, "image/obj", PrepareObjectID(id, "default").Id)
	assert.Equal(t, "image/bare", PrepareObjectID(&ObjectID{Id: "bare"}, "image").Id)
}

func TestDefaultGroupFromURL(t *testing.T) {
	parse := func(raw string) *url.URL {
		u, err := url.Parse(raw)
		require.NoError(t, err)
		return u
	}
	assert.Equal(t, "", defaultGroupFromURL(parse("apfs://apfs:8081")))
	assert.Equal(t, "", defaultGroupFromURL(parse("tcp://host:8081")))
	assert.Equal(t, "", defaultGroupFromURL(parse("apfs://apfs:8081/")))
	assert.Equal(t, "image", defaultGroupFromURL(parse("tcp://host:8081/image")))
	assert.Equal(t, "video", defaultGroupFromURL(parse("tcp://host:8081/?group=video")))
	assert.Equal(t, "video", defaultGroupFromURL(parse("tcp://host:8081/image?group=video")))
	assert.Equal(t, "", defaultGroupFromURL(nil))
}
