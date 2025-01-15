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

func updateLocker(conf *appcontext.ConfigType) api.UpdateStateFnk {
	conn := conf.Storage.ProcessingInterlockConnect
	switch {
	case strings.HasPrefix(conn, "redis://"):
		return redisLocker(conn, conf.Storage.ProcessingLifetime)
	case conn == "memory" || conn == "":
		return lruLocker(conf.Storage.ProcessingLifetime)
	default:
		panic(fmt.Errorf("invalid interlock option: %s", conf.Storage.ProcessingInterlockConnect))
	}
}

func redisLocker(conn string, lifetime time.Duration) api.UpdateStateFnk {
	rlock, err := redislock.NewByURL(conn, lifetime)
	if err != nil {
		log.Fatal(err)
	}
	return func(key any) bool {
		return rlock.TryLock(key) != nil
	}
}

func lruLocker(lifetime time.Duration) api.UpdateStateFnk {
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
