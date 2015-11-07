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
	"errors"
	"time"
)

var (
	ErrLimitReached = errors.New("Limit reached")
)

type TokenBucket struct {
	Used           float64
	LastAccessTime time.Time
	Limit          float64
	Duration       time.Duration
}

func NewTokenBucket(limit float64, duration time.Duration) *TokenBucket {
	return &TokenBucket{0, time.Now(), limit, duration}
}

func (bucket *TokenBucket) Consume(count float64) error {
	now := time.Now()
	used := bucket.GetAdjustedUsage(now)

	if used+count <= bucket.Limit {
		bucket.Used = used + count
		bucket.LastAccessTime = now
		return nil
	}

	return ErrLimitReached
}

func (bucket *TokenBucket) GetAdjustedUsage(now time.Time) float64 {
	used := bucket.Used
	if bucket.LastAccessTime.Unix() > 0 {
		elapsed := now.Sub(bucket.LastAccessTime)
		back := bucket.Limit * float64(elapsed) / float64(bucket.Duration)
		used -= back
		if used < 0 {
			used = 0
		}
	}
	return used
}
