package reader

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/go-redis/redis/v9"
	"github.com/jaredmcqueen/tsdb-writer/util"
)

var (
	ReaderChan = make(chan map[string]interface{})
)

// RedisStreamReader reads from multiple streams, keeping point-in-times for each
// all message.Values are sent to a channel without modification
func RedisStreamsReader(pubsub *util.Pubsub) {
	ctx := context.Background()
	log.Println("connecting to redis streams endpoint", util.Config.RedisStreamsEndpoint)
	rdb := redis.NewClient(&redis.Options{
		Addr: util.Config.RedisStreamsEndpoint,
	})

	// test redis connection
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		log.Fatal("could not connect to redis", err)
	}
	log.Println("connected to redis")

	pitMap := make(map[string]string)
	for _, streamName := range strings.Split(util.Config.RedisStreamsNames, " ") {
		pitMap[streamName] = util.Config.RedisStreamsStart
	}

	newpits := func() []string {
		streams := []string{}
		pits := []string{}
		for s, p := range pitMap {
			streams = append(streams, s)
			pits = append(pits, p)
		}
		return append(streams, pits...)
	}

	for {
		items, err := rdb.XRead(ctx,
			&redis.XReadArgs{
				Streams: newpits(),
				Count:   util.Config.RedisStreamsCount,
				Block:   time.Duration(time.Second),
			},
		).Result()
		if err != nil {
			log.Println("error XRead: ", err)
		}
		for _, stream := range items {
			for _, message := range stream.Messages {
				pubsub.Publish("redis", message.Values)
				pitMap[stream.Stream] = message.ID
			}
		}
	}
}
