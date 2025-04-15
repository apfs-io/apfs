package version

import (
	"context"
	"fmt"

	"github.com/demdxx/gocast/v2"
)

var ctxVersionKey = &struct{ s string }{"version"}

type Version struct {
	Version string
	Commit  string
	Date    string
}

func (v *Version) String() string {
	return fmt.Sprintf("%s-%s / %s", v.Version, v.Commit, v.Date)
}

func (v *Version) Public() string {
	if v.IsEmpty() {
		return "unknown"
	}
	if v.Commit == "" {
		return v.Version
	}
	if v.Version == "" {
		return v.Commit
	}
	return fmt.Sprintf("%s-%s", v.Version, v.Commit)
}

func (v *Version) IsEmpty() bool {
	return v == nil || (v.Version == "" && v.Commit == "")
}

func WithContext(ctx context.Context, version *Version) context.Context {
	return context.WithValue(ctx, ctxVersionKey, version)
}

func Get(ctx context.Context) *Version {
	version, _ := ctx.Value(ctxVersionKey).(*Version)
	return gocast.IfThen(version != nil, version, &Version{})
}
