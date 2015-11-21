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
	"bytes"
	"encoding/gob"
	"errors"
	redis "github.com/streamrail/redis-storage"
	"math"
	"strings"
	"time"
)

var (
	ErrNotFound     = errors.New("Not found")
	ErrKeyEmpty     = errors.New("Key cannot be empty")
	ErrCountZero    = errors.New("Count should be greater than zero")
	ErrLimitZero    = errors.New("Limit should be greater than zero")
	ErrCountLimit   = errors.New("Limit should be greater than count")
	ErrZeroDuration = errors.New("Duration cannot be zero")
)

type Limiter interface {
	Get(key string) (int64, error)
	Post(key string, count int64, limit int64, duration time.Duration) (int64, error)
	Delete(key string) error
}

type SingleThreadLimiter struct {
	storage  *redis.RedisStorage
	reqChan  chan request
	stopChan chan int
}

func NewSingleThreadLimiter(storage *redis.RedisStorage) *SingleThreadLimiter {
	return &SingleThreadLimiter{storage, make(chan request), make(chan int)}
}

func (l *SingleThreadLimiter) Start() {
	go l.serve()
}

func (l *SingleThreadLimiter) Stop() {
	l.stopChan <- 1
}

func (l *SingleThreadLimiter) Kill(key string) {
	l.Stop()
	l.Delete(key)
}

func (l *SingleThreadLimiter) Post(key string, count, limit int64, duration time.Duration) (int64, error) {

	err := checkPostArgs(key, count, limit, duration)

	if err != nil {
		return 0, err
	}

	req := request{
		POST,
		key,
		count,
		limit,
		duration,
		make(chan response),
	}
	l.reqChan <- req
	res := <-req.response
	return res.used, res.err
}

func (l *SingleThreadLimiter) Get(key string) (int64, error) {
	req := request{
		GET,
		key,
		0,
		0,
		0,
		make(chan response),
	}
	l.reqChan <- req
	res := <-req.response
	return res.used, res.err
}

func (l *SingleThreadLimiter) Delete(key string) error {
	req := request{
		DELETE,
		key,
		0,
		0,
		0,
		make(chan response),
	}
	l.reqChan <- req
	res := <-req.response
	return res.err
}

func (l *SingleThreadLimiter) serve() {
	for {
		select {
		case _ = <-l.stopChan:
			break
		case req := <-l.reqChan:
			switch req.method {
			case GET:
				bucket, err := l.GetTokenBucket(req.key)
				if err != nil {
					//log.Println("GET/GetTokenBucket error: " + err.Error())
					req.response <- response{0, err}
					continue
				}
				now := time.Now()
				req.response <- response{usage(bucket.GetAdjustedUsage(now)), nil}
			case DELETE:
				err := l.storage.Delete(req.key)
				if err != nil {
					//log.Printf("error deleting key %s: %s\n", req.key, err.Error())
				} else {
					//log.Printf("key %s deleted\n", req.key)
				}
				req.response <- response{0, err}
			case POST:
				bucket, err := l.GetTokenBucket(req.key)
				if err != nil && err != redis.ErrNil {
					//log.Println("POST/GetTokenBucket error: " + err.Error())
					req.response <- response{0, err}
					continue
				}

				count, limit := float64(req.count), float64(req.limit)
				duration := req.duration
				if bucket == nil {
					bucket = NewTokenBucket(limit, duration)
				} else {
					if bucket.Limit != limit || bucket.Duration != duration {
						bucket = NewTokenBucket(limit, duration)
					}
				}

				err = bucket.Consume(count)
				if err != nil {
					//log.Println("bucket.Consume error: " + err.Error())
					req.response <- response{usage(bucket.Used), err}
					continue
				}
				//log.Printf("Setex token bucket: key=%s\n", req.key)
				err = l.storage.Setex(req.key, bucket, duration)
				if err != nil {
					//log.Println(err)
					req.response <- response{0, err}
					continue
				}
				req.response <- response{usage(bucket.Used), nil}
			default:
				req.response <- response{0, errors.New("Undefined Method")}
				continue
			}
		}
	}
}

func checkPostArgs(key string, count, limit int64, duration time.Duration) error {
	switch true {
	case len(strings.TrimSpace(key)) == 0:
		return ErrKeyEmpty
	case count <= 0:
		return ErrCountZero
	case limit <= 0:
		return ErrLimitZero
	case count > limit:
		return ErrCountLimit
	case duration == 0:
		return ErrZeroDuration
	}
	return nil
}

type response struct {
	used int64
	err  error
}

const (
	GET = iota
	POST
	DELETE
)

type request struct {
	method   int
	key      string
	count    int64
	limit    int64
	duration time.Duration
	response chan response
}

func usage(f float64) int64 {
	return int64(math.Ceil(f))
}

func (l *SingleThreadLimiter) GetTokenBucket(key string) (*TokenBucket, error) {
	//log.Printf("GetTokenBucket: key = %s\n", key)
	data, err := l.storage.Get(key)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
	}

	var result = new(TokenBucket)
	dec := gob.NewDecoder(bytes.NewBuffer(data))
	err = dec.Decode(result)

	if err != nil {
		//log.Println(err.Error())
		return nil, err
	}
	return result, nil
}
