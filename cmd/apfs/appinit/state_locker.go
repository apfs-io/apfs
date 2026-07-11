package appinit

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/demdxx/gocast/v2"
	"github.com/demdxx/interlock/redislock"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/pkg/errors"

	"github.com/apfs-io/apfs/cmd/apfs/appcontext"
	api "github.com/apfs-io/apfs/internal/server/v1"
)

func updateLocker(conf *appcontext.ProcessingConfig) api.UpdateStateFunc {
	conn := conf.InterlockConnect
	switch {
	case strings.HasPrefix(conn, "redis://"):
		return redisLocker(conn, conf.Lifetime)
	case conn == "memory" || conn == "":
		return lruLocker(conf.Lifetime)
	default:
		panic(fmt.Errorf("invalid interlock option: %s", conf.InterlockConnect))
	}
}

func redisLocker(conn string, lifetime time.Duration) api.UpdateStateFunc {
	rlock, err := redislock.NewByURL(conn, lifetime)
	if err != nil {
		log.Fatal(err)
	}
	return func(key any) bool {
		return rlock.TryLock(key) != nil
	}
}

func lruLocker(lifetime time.Duration) api.UpdateStateFunc {
	cache, err := lru.New[string, any](1024)
	if err != nil {
		panic(errors.Wrap(err, `init LRU cache`))
	}
	return func(key any) bool {
		skey := gocast.Str(key)
		tm, ok := cache.Get(skey)
		if !ok || tm == nil || time.Since(tm.(time.Time)) > lifetime {
			cache.Add(skey, time.Now())
			return true
		}
		return false
	}
}
