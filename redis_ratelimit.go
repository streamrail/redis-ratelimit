// Copyright (c) 2015 streamrail

// The MIT License (MIT)

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package ratelimit

import (
	"fmt"
	redis "github.com/streamrail/redis-storage"
	"time"
)

type Ratelimit struct {
	limit    int64
	duration time.Duration
	limiter  *SingleThreadLimiter
	rs       *redis.RedisStorage
}

func NewRatelimit(limit int64, duration time.Duration, redisHost string,
	redisConnPoolSize int, redisPrefix string) *Ratelimit {
	return &Ratelimit{
		limit:    limit,
		duration: duration,
		rs:       redis.NewRedisStorage(redisHost, redisConnPoolSize, redisPrefix),
	}
}

func (r *Ratelimit) Start() {
	r.limiter = NewSingleThreadLimiter(r.rs)
	r.limiter.Start()
}

func (r *Ratelimit) Stop() error {
	if r.limiter == nil {
		return fmt.Errorf("cannot stop nil rate limiter")
	}
	r.limiter.Stop()
	return nil
}

func (r *Ratelimit) Incr(key string) (int64, error) {
	return r.limiter.Post(key, 1, r.limit, r.duration)
}
