package objectpath

import (
	"strings"
	"testing"
	"time"
)

func TestGenerator(t *testing.T) {
	pattern := "{{year}}/{{month}}/{{day}}/hash/{{md5:1}}/{{md5:2}}/{{md5:3}}/{{md5}}"
	gen := NewBasePathgenerator(pattern, WithChecker(func(path string) bool { return true }))
	targetPrefix := time.Now().Format("2006/01/02/hash/")
	for i := 0; i < 10; i++ {
		newPath, err := gen.Generate("prefix")
		if err != nil {
			t.Error(err)
		}
		if !strings.HasPrefix(newPath, targetPrefix) {
			t.Error("invalid path generation")
		}
		if len(newPath) <= len(targetPrefix)+4 {
			t.Error("invalid path size generation")
		}
	}
}
