package register

import (
	"github.com/go-redis/redis"
	"time"
	"puppy/config"
	"fmt"
	"log"
	"sync"
)
var mux sync.Mutex

type RedisRegister struct {
	Register
	RedisHost string
	TtlSecs   int
	client    *redis.Client
	srvMap* map[string] []string
}

func activate(m *map[string] []string){
	mux.Lock()
	Reg.(*RedisRegister).srvMap=m
	mux.Unlock()
}
func (r *RedisRegister) TrackingServices() {
	keys, err := r.client.Keys("*").Result()
	if (err != nil) {
		log.Fatalln(err)
		return
	}
	now := time.Now().Unix()
	min := now - int64(config.Instance.RegisterInfoTTL)
	pipe := r.client.Pipeline()
	for _, key := range keys {
		pipe.ZRangeByScore(key, redis.ZRangeBy{Min: fmt.Sprintf("%d", min), Max: fmt.Sprintf("%d", now)})
	}
	cmds, err := pipe.Exec()
	if (err != nil) {
		log.Fatalln(err)
		return
	}
	srvMap:=make(map[string][]string,0)
	appendResult := func(key string, hosts []string) {
		for _, host := range hosts {
			srvMap[key] = append(srvMap[key], host)
		}
	}

	var result []string
	for _, c := range cmds {
		switch c.Name() {
		case "zrangebyscore":
			key := c.Args()[1].(string)
			result, err = c.(*redis.StringSliceCmd).Result()
			if err != nil {
				log.Fatalln(err)
			} else {
				appendResult(key, result)
			}
		}
	}
	activate(&srvMap)
}

func (r *RedisRegister) QueryMethod(methodSign string) ([]string, error) {
	if(r.srvMap==nil){
		return []string{},nil
	}
	if ret,ok:=(*r.srvMap)[methodSign];ok{
		return ret,nil
	}
	return []string{},nil
}

func (r *RedisRegister) RegisterMethod(methods []string, host string, weight int) error {
	now := time.Now().Unix()
	pipe := r.client.Pipeline()
	for _, method := range methods {
		pipe.ZAdd(method, redis.Z{float64(now), host})
	}
	_, err := pipe.Exec()
	return err
}

func (r *RedisRegister) Init() *RedisRegister {
	m:=make(map[string] []string,0)
	r.srvMap=&m
	var cfg=config.Instance.Register
	r.client = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d",cfg.Host,cfg.Port),
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	Reg = r
	return r
}
