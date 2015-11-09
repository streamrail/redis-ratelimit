# redis-ratelimit (WIP)

## Summary

Simple rate limiter for your golang app. Basically code extraction from [ratelimitd](https://github.com/ctulek/ratelimit) to allow usage inside custom golang apps, not as a daemon.

## Usage

```go
package api

import (
	"flag"
	ratelimit "github.com/streamrail/redis-ratelimit"
	"net/http"
	"time"
)

var (
	redisHost         = flag.String("ratelimit_redis", "localhost:6379", "Redis host and port. Eg: localhost:6379")
	redisConnPoolSize = flag.Int("ratelimit_redisConnPoolSize", 5, "Redis connection pool size. Default: 5")
	redisPrefix       = flag.String("ratelimit_redisPrefix", "rl_", "Redis prefix to attach to keys")

	ipRateLimiter     = ratelimit.NewRatelimit(1, 10*time.Second, *redisHost, *redisConnPoolSize, *redisPrefix)
)

func init() {
	ipRateLimiter.Start()
}

func GetApiResponse(w http.ResponseWriter, key string, r *http.Request) []byte {
	if _, err := ipRateLimiter.Incr(util.GetIP(r)); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return []byte(err.Error())
	} 
...
...
...
```


## TODO
- add tests
- create examples

## License
[Original license](https://raw.githubusercontent.com/ctulek/ratelimit/master/LICENSE) included, plus additional MIT license we added