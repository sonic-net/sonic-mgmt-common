package db

import (
	"flag"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/go-redis/redis/v7"
	log "github.com/golang/glog"
)

var usePools = flag.Bool("use_connection_pools", true, "use connection pools for Redis Clients")

const (
	POOL_SIZE = 25
)

var rcm *redisClientManager
var initMu = &sync.Mutex{}

type redisClientManager struct {
	// clients holds one Redis Client for each DBNum
	clients                            [MaxDB + 1]*redis.Client
	mu                                 *sync.RWMutex
	curTransactionalClients            atomic.Int32
	totalPoolClientsRequested          atomic.Uint64
	totalTransactionalClientsRequested atomic.Uint64
}

type RedisCounters struct {
	CurTransactionalClients            uint32                      // The number of transactional clients currently opened.
	TotalPoolClientsRequested          uint64                      // The total number of Redis Clients using a connection pool requested.
	TotalTransactionalClientsRequested uint64                      // The total number of Transactional Redis Clients requested.
	PoolStatsPerDB                     map[string]*redis.PoolStats // The pool counters for each Redis Client in the cache.
}

func init() {
	initializeRedisOpts()
	initializeRedisClientManager()
}

func initializeRedisClientManager() {
	initMu.Lock()
	defer initMu.Unlock()
	if rcm != nil {
		return
	}
	rcm = &redisClientManager{
		clients:                            [MaxDB + 1]*redis.Client{},
		mu:                                 &sync.RWMutex{},
		curTransactionalClients:            atomic.Int32{},
		totalPoolClientsRequested:          atomic.Uint64{},
		totalTransactionalClientsRequested: atomic.Uint64{},
	}
	rcm.mu.Lock()
	defer rcm.mu.Unlock()
	for dbNum := DBNum(0); dbNum < MaxDB; dbNum++ {
		if len(getDBInstName(dbNum)) == 0 {
			continue
		}
		// Create a Redis Client for each database.
		rcm.clients[int(dbNum)] = createRedisClient(dbNum, POOL_SIZE)
	}
}

func createRedisClient(db DBNum, poolSize int) *redis.Client {
	opts := adjustRedisOpts(&Options{DBNo: db})
	opts.PoolSize = poolSize
	client := redis.NewClient(opts)
	if _, err := client.Ping().Result(); err != nil {
		log.V(0).Infof("RCM error during Redis Client creation for DBNum=%v: %v", db, err)
	}
	return client
}

func createRedisClientWithOpts(opts *redis.Options) *redis.Client {
	client := redis.NewClient(opts)
	if _, err := client.Ping().Result(); err != nil {
		log.V(0).Infof("RCM error during Redis Client creation for DBNum=%v: %v", opts.DB, err)
	}
	return client
}

func getClient(db DBNum) *redis.Client {
	rcm.mu.RLock()
	defer rcm.mu.RUnlock()
	return rcm.clients[int(db)]
}

// RedisClient will return a Redis Client that can be used for non-transactional Redis operations.
// The client returned by this function is shared among many DB readers/writers and uses
// a connection pool. For transactional Redis operations, please use GetRedisClientForTransaction().
func RedisClient(db DBNum) *redis.Client {
	if rcm == nil {
		initializeRedisClientManager()
	}
	if !(*usePools) { // Connection Pooling is disabled.
		return TransactionalRedisClient(db)
	}
	if len(getDBInstName(db)) == 0 {
		log.V(0).Infof("Invalid DBNum requested: %v", db)
		return nil
	}
	rcm.totalPoolClientsRequested.Add(1)
	rc := getClient(db)
	if rc == nil {
		log.V(0).Infof("RCM Redis client for DBNum=%v is nil!", db)
		rcm.mu.Lock()
		defer rcm.mu.Unlock()
		if rc = rcm.clients[int(db)]; rc != nil {
			return rc
		}
		rc = createRedisClient(db, POOL_SIZE)
		rcm.clients[int(db)] = rc
	}
	return rc
}

// TransactionalRedisClient will create and return a unique Redis client. This client can be used
// for transactional operations. These operations include MULTI, PSUBSCRIBE (PubSub), and SCAN. This
// client must be closed using CloseRedisClient when it is no longer needed.
func TransactionalRedisClient(db DBNum) *redis.Client {
	if rcm == nil {
		initializeRedisClientManager()
	}
	if len(getDBInstName(db)) == 0 {
		log.V(0).Infof("Invalid DBNum requested: %v", db)
		return nil
	}
	rcm.totalTransactionalClientsRequested.Add(1)
	client := createRedisClient(db, 1)
	rcm.curTransactionalClients.Add(1)
	return client
}

func TransactionalRedisClientWithOpts(opts *redis.Options) *redis.Client {
	if rcm == nil {
		initializeRedisClientManager()
	}
	rcm.totalTransactionalClientsRequested.Add(1)
	opts.PoolSize = 1
	client := createRedisClientWithOpts(opts)
	rcm.curTransactionalClients.Add(1)
	return client
}

// CloseUniqueRedisClient will close the Redis client that is passed in.
func CloseRedisClient(rc *redis.Client) error {
	if rcm == nil {
		return fmt.Errorf("RCM is nil when trying to close Redis Client: %v", rc)
	}
	if rc == nil {
		return nil
	}
	// Closing a Redis Client with a connection pool is a no-op because these clients need to stay open.
	if !IsTransactionalClient(rc) {
		return nil
	}
	if err := rc.Close(); err != nil {
		return err
	}
	rcm.curTransactionalClients.Add(-1)
	return nil
}

// IsTransactionalClient returns true if rc is a transactional client and false otherwise.
func IsTransactionalClient(rc *redis.Client) bool {
	if rc == nil {
		return false
	}
	return rc.Options().PoolSize == 1
}

// RedisClientManagerCounters returns the counters stored in the RCM.
func RedisClientManagerCounters() *RedisCounters {
	if rcm == nil {
		initializeRedisClientManager()
	}
	counters := &RedisCounters{
		CurTransactionalClients:            uint32(rcm.curTransactionalClients.Load()),
		TotalPoolClientsRequested:          rcm.totalPoolClientsRequested.Load(),
		TotalTransactionalClientsRequested: rcm.totalTransactionalClientsRequested.Load(),
		PoolStatsPerDB:                     map[string]*redis.PoolStats{},
	}
	rcm.mu.RLock()
	defer rcm.mu.RUnlock()
	for db, client := range rcm.clients {
		dbName := getDBInstName(DBNum(db))
		if dbName == "" || client == nil {
			continue
		}
		counters.PoolStatsPerDB[dbName] = client.PoolStats()
	}
	return counters
}
