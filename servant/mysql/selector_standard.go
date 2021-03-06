package mysql

import (
	"github.com/funkygao/fae/config"
	"github.com/funkygao/golib/cache"
	"github.com/funkygao/golib/str"
	log "github.com/funkygao/log4go"
	"strconv"
	"strings"
	"time"
)

type StandardServerSelector struct {
	conf    *config.ConfigMysql
	clients map[string]*mysql // key is pool

	lookupCache *cache.LruCache // {pool+hintId: *mysql}
}

func newStandardServerSelector(cf *config.ConfigMysql) (this *StandardServerSelector) {
	this = new(StandardServerSelector)
	this.conf = cf
	this.lookupCache = cache.NewLruCache(cf.LookupCacheMaxItems)
	this.clients = make(map[string]*mysql)
	for _, server := range cf.Servers {
		my := newMysql(server.DSN(), cf.CachePrepareStmtMaxItems, &cf.Breaker)
		for retries := uint(0); retries < cf.Breaker.FailureAllowance; retries++ {
			log.Debug("mysql connecting: %s", server.DSN())

			if my.Open() == nil && my.Ping() == nil {
				// sql.Open() does not establish any connections to the database
				// sql.Ping() does
				break
			}

			my.breaker.Fail()
		}

		if !my.breaker.Open() {
			my.db.SetMaxIdleConns(cf.MaxIdleConnsPerServer)
			// TODO https://code.google.com/p/go/source/detail?r=8a7ac002f840
			my.db.SetMaxOpenConns(cf.MaxConnsPerServer)
			this.clients[server.Pool] = my

			if cf.MaxIdleTime > 0 {
				go func() {
					for _ = range time.Tick(cf.MaxIdleTime) {
						if err := my.Ping(); err != nil {
							log.Error("mysql[%s]: %s", server.DSN(), err.Error())
							my.breaker.Fail()
						}

					}
				}()
			}
		}

	}

	return
}

func (this *StandardServerSelector) PickServer(pool string,
	table string, hintId int) (*mysql, error) {
	if this.shardedPool(pool) {
		return this.pickShardedServer(pool, table, hintId)
	}

	return this.pickNonShardedServer(pool, table)
}

func (this *StandardServerSelector) ServerByBucket(bucket string) (*mysql, error) {
	my, present := this.clients[bucket]
	if !present {
		return nil, ErrServerNotFound
	}

	return my, nil
}

func (this *StandardServerSelector) Servers() []*mysql {
	r := make([]*mysql, 0)
	for _, m := range this.clients {
		r = append(r, m)
	}

	return r
}

func (this *StandardServerSelector) PoolServers(pool string) []*mysql {
	r := make([]*mysql, 0)
	for p, my := range this.clients {
		// p=UserShard2 pool=UserShard
		if strings.HasPrefix(p, pool) {
			r = append(r, my)
		}
	}
	return r
}

func (this *StandardServerSelector) shardedPool(pool string) bool {
	if _, present := this.conf.GlobalPools[pool]; present {
		return false
	}

	return true
}

func (this *StandardServerSelector) KickLookupCache(pool string, hintId int) {
	if pool != this.conf.LookupPool || hintId == 0 {
		return
	}

	key := this.lookupCacheKey(pool, hintId)
	this.lookupCache.Del(key)
	log.Trace("lookupCache[%s] kicked", key)
}

func (this *StandardServerSelector) lookupCacheKey(pool string, hintId int) string {
	// FIXME how to handle cache kick?
	// TODO itoa is too slow, 143 ns/op, use int as cache key
	return pool + ":" + strconv.Itoa(hintId)
}

func (this *StandardServerSelector) pickShardedServer(pool string,
	table string, hintId int) (*mysql, error) {
	const (
		sb1 = "SELECT shardId,shardLock FROM "
		sb2 = " WHERE entityId=?"
	)

	if hintId == 0 {
		return nil, ErrInvalidHintId
	}

	// get mysql conn from cache
	key := this.lookupCacheKey(pool, hintId)
	if conn, present := this.lookupCache.Get(key); present {
		return conn.(*mysql), nil
	}

	// cache missed, get lookup mysql conn
	my, err := this.ServerByBucket(this.conf.LookupPool)
	if err != nil {
		return nil, err
	}

	// TODO maybe string concat is better performant
	sb := str.NewStringBuilder()
	sb.WriteString(sb1)
	lookupTable := this.conf.LookupTable(pool)
	if lookupTable == "" {
		return nil, ErrLookupTableNotFound
	}
	sb.WriteString(lookupTable)
	sb.WriteString(sb2)
	rows, err := my.Query(sb.String(), hintId)
	if err != nil {
		log.Error("sql=%s id=%d: %s", sb.String(), hintId, err.Error())
		return nil, err
	}

	defer rows.Close()

	// only 1 row in lookup table
	if !rows.Next() {
		return nil, ErrShardLookupNotFound
	}

	var (
		shardId     string
		shardLocked int
	)
	if err = rows.Scan(&shardId, &shardLocked); err != nil {
		log.Error("sql=%s id=%d: %s", sb.String(), hintId, err.Error())
		return nil, err
	}
	if shardLocked > 0 {
		return nil, ErrEntityLocked
	}
	if err = rows.Err(); err != nil {
		log.Error("sql=%s id=%d: %s", sb.String(), hintId, err.Error())
		return nil, err
	}

	//bucket := fmt.Sprintf("%s%d", pool, (hintId/200000)+1)
	bucket := pool + shardId
	my, present := this.clients[bucket]
	if !present {
		return nil, ErrServerNotFound
	}

	this.lookupCache.Set(key, my)
	log.Debug("lookupCache[%s] set {pool^%s}", key, bucket)

	return my, nil
}

func (this *StandardServerSelector) pickNonShardedServer(pool string,
	table string) (*mysql, error) {
	my, present := this.clients[pool]
	if !present {
		return nil, ErrServerNotFound
	}

	return my, nil
}

func (this *StandardServerSelector) endsWithDigit(pool string) bool {
	lastChar := pool[len(pool)-1]
	return lastChar >= '0' && lastChar <= '9'
}
