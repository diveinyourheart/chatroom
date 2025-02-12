package main

import (
	"time"

	"github.com/gomodule/redigo/redis"
)

var pl *redis.Pool

func initPool(address string, maxchanId, maxAc int, idTO time.Duration) {
	pl = &redis.Pool{
		MaxIdle:     maxchanId,
		MaxActive:   maxAc,
		IdleTimeout: idTO,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", address)
		},
	}
}
