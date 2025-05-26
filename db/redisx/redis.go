package redisx

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/wjoj/tool/v2/log"
	"github.com/wjoj/tool/v2/utils"
)

type Config struct {
	Addrs           []string      `json:"addrs" yaml:"addrs"`
	IsCluster       bool          `json:"isCluster" yaml:"isCluster"`
	Username        string        `json:"username" yaml:"username"`
	Password        string        `json:"password" yaml:"password"`
	ReadTimeout     time.Duration `yaml:"readTimeout"`
	WriteTimeout    time.Duration `yaml:"writeTimeout"`
	PoolSize        int           `json:"poolSize" yaml:"poolSize"`
	MinIdleConns    int           `json:"minIdleConns" yaml:"minIdleConns"`
	MaxConnAge      time.Duration `json:"maxConnAge" yaml:"maxConnAge"`
	MaxIdleConns    int           `yaml:"maxIdleConns"`   //最大空闲连接数。默认0
	MaxActiveConns  int           `yaml:"maxActiveConns"` //
	PoolTimeout     time.Duration `json:"poolTimeout" yaml:"poolTimeout"`
	IdleTimeout     time.Duration `json:"idleTimeout" yaml:"idleTimeout"`
	ConnMaxIdleTime time.Duration `yaml:"connMaxIdleTime"`
	ConnMaxLifetime time.Duration `yaml:"connMaxLifetime"`
}

type ClientInf interface {
	redis.Cmdable
	Close() error
}

type Clientx struct {
	ClientInf
	cfg *Config
}

func New(cfg *Config) (*Clientx, error) {
	if len(cfg.Addrs) == 0 {
		return nil, fmt.Errorf("redis adds can't be empty")
	}
	if cfg.ReadTimeout > 0 {
		cfg.ReadTimeout *= time.Second
	}
	if cfg.WriteTimeout > 0 {
		cfg.WriteTimeout *= time.Second
	}
	var cli ClientInf
	if cfg.IsCluster {
		cli = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:           cfg.Addrs,
			Password:        cfg.Password,
			ReadTimeout:     cfg.ReadTimeout,
			WriteTimeout:    cfg.WriteTimeout,
			PoolSize:        cfg.PoolSize,
			PoolTimeout:     cfg.PoolTimeout * time.Second,
			MinIdleConns:    cfg.MinIdleConns,
			MaxIdleConns:    cfg.MaxIdleConns,
			MaxActiveConns:  cfg.MaxActiveConns,
			ConnMaxIdleTime: cfg.ConnMaxIdleTime * time.Second,
			ConnMaxLifetime: cfg.ConnMaxIdleTime * time.Second,
		})
	} else {
		cli = redis.NewClient(&redis.Options{
			Addr:            cfg.Addrs[0],
			Password:        cfg.Password,
			ReadTimeout:     cfg.ReadTimeout,
			WriteTimeout:    cfg.WriteTimeout,
			PoolSize:        cfg.PoolSize,
			PoolTimeout:     cfg.PoolTimeout * time.Second,
			MinIdleConns:    cfg.MinIdleConns,
			MaxIdleConns:    cfg.MaxIdleConns,
			MaxActiveConns:  cfg.MaxActiveConns,
			ConnMaxIdleTime: cfg.ConnMaxIdleTime * time.Second,
			ConnMaxLifetime: cfg.ConnMaxIdleTime * time.Second,
		})
	}
	if _, err := cli.Ping(context.Background()).Result(); err != nil {
		return nil, err
	}
	return &Clientx{
		ClientInf: cli,
		cfg:       cfg,
	}, nil
}

var rd *Clientx
var rdMap map[string]*Clientx
var defaultKey = utils.DefaultKey.DefaultKey

func Init(cfgs map[string]Config, options ...Option) error {
	log.Info("init redis")
	opt := applyGenGormOptions(options...)
	defaultKey = opt.defKey.DefaultKey
	rdMap = make(map[string]*Clientx)
	if len(opt.defKey.Keys) != 0 {
		opt.defKey.Keys = append(opt.defKey.Keys, opt.defKey.DefaultKey)
		for _, key := range opt.defKey.Keys {
			_, is := rdMap[key]
			if is {
				continue
			}
			cfg, is := cfgs[key]
			if !is {
				log.Errorf("init redis client %s not found", key)
				return fmt.Errorf("redis client %s not found", key)
			}
			cli, err := New(&cfg)
			if err != nil {
				log.Errorf("init redis client %s init error: %v", key, err)
				return err
			}
			rdMap[key] = cli
			if key == defaultKey {
				rd = cli
			}
		}
		log.Info("init redis success")
		return nil
	}
	for name, cfg := range cfgs {
		cli, err := New(&cfg)
		if err != nil {
			log.Errorf("init redis client %s init error: %v", name, err)
			return err
		}
		rdMap[name] = cli
		if name == defaultKey {
			rd = cli
		}
	}
	log.Info("init redis success")
	return nil
}

func InitGlobal(cfg *Config) error {
	var err error
	rd, err = New(cfg)
	if err != nil {
		return err
	}
	return nil
}

func GetClient(name ...string) *Clientx {
	if len(name) == 0 {
		cli, is := rdMap[defaultKey]
		if !is {
			panic(fmt.Errorf("redis client %s not found", utils.DefaultKey.DefaultKey))
		}
		return cli
	}
	cli, is := rdMap[name[0]]
	if !is {
		panic(fmt.Errorf("redis client %s not found", name[0]))
	}
	return cli
}

// Client 重新一遍所有方法
func Client() *Clientx {
	return rd
}

func GetConfig() Config {
	return *rd.cfg
}

func Close() error {
	return rd.Close()
}

func CloseAll() error {
	for _, cli := range rdMap {
		cli.Close()
	}
	return nil
}

// Key 相关操作
func Del(ctx context.Context, keys ...string) *redis.IntCmd {
	return rd.Del(ctx, keys...)
}

func Exists(ctx context.Context, keys ...string) *redis.IntCmd {
	return rd.Exists(ctx, keys...)
}

func Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd {
	return rd.Expire(ctx, key, expiration)
}

func ExpireAt(ctx context.Context, key string, tm time.Time) *redis.BoolCmd {
	return rd.ExpireAt(ctx, key, tm)
}

func Keys(ctx context.Context, pattern string) *redis.StringSliceCmd {
	return rd.Keys(ctx, pattern)
}

func Persist(ctx context.Context, key string) *redis.BoolCmd {
	return rd.Persist(ctx, key)
}

func PExpire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd {
	return rd.PExpire(ctx, key, expiration)
}

func PExpireAt(ctx context.Context, key string, tm time.Time) *redis.BoolCmd {
	return rd.PExpireAt(ctx, key, tm)
}

func PTTL(ctx context.Context, key string) *redis.DurationCmd {
	return rd.PTTL(ctx, key)
}

func Rename(ctx context.Context, key, newkey string) *redis.StatusCmd {
	return rd.Rename(ctx, key, newkey)
}

func RenameNX(ctx context.Context, key, newkey string) *redis.BoolCmd {
	return rd.RenameNX(ctx, key, newkey)
}

func TTL(ctx context.Context, key string) *redis.DurationCmd {
	return rd.TTL(ctx, key)
}

func Type(ctx context.Context, key string) *redis.StatusCmd {
	return rd.Type(ctx, key)
}

// String 相关操作
func Get(ctx context.Context, key string) *redis.StringCmd {
	return rd.Get(ctx, key)
}

func GetRange(ctx context.Context, key string, start, end int64) *redis.StringCmd {
	return rd.GetRange(ctx, key, start, end)
}

func GetSet(ctx context.Context, key string, value interface{}) *redis.StringCmd {
	return rd.GetSet(ctx, key, value)
}

func Incr(ctx context.Context, key string) *redis.IntCmd {
	return rd.Incr(ctx, key)
}

func IncrBy(ctx context.Context, key string, value int64) *redis.IntCmd {
	return rd.IncrBy(ctx, key, value)
}

func IncrByFloat(ctx context.Context, key string, value float64) *redis.FloatCmd {
	return rd.IncrByFloat(ctx, key, value)
}

func MGet(ctx context.Context, keys ...string) *redis.SliceCmd {
	return rd.MGet(ctx, keys...)
}

func MSet(ctx context.Context, values ...interface{}) *redis.StatusCmd {
	return rd.MSet(ctx, values...)
}

func MSetNX(ctx context.Context, values ...interface{}) *redis.BoolCmd {
	return rd.MSetNX(ctx, values...)
}

func Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	return rd.Set(ctx, key, value, expiration)
}

func SetEx(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	return rd.SetEx(ctx, key, value, expiration)
}

// List 相关操作
func LIndex(ctx context.Context, key string, index int64) *redis.StringCmd {
	return rd.LIndex(ctx, key, index)
}

func LInsert(ctx context.Context, key, op string, pivot, value interface{}) *redis.IntCmd {
	return rd.LInsert(ctx, key, op, pivot, value)
}

func LInsertBefore(ctx context.Context, key string, pivot, value interface{}) *redis.IntCmd {
	return rd.LInsertBefore(ctx, key, pivot, value)
}

func LInsertAfter(ctx context.Context, key string, pivot, value interface{}) *redis.IntCmd {
	return rd.LInsertAfter(ctx, key, pivot, value)
}

func LLen(ctx context.Context, key string) *redis.IntCmd {
	return rd.LLen(ctx, key)
}

func LPop(ctx context.Context, key string) *redis.StringCmd {
	return rd.LPop(ctx, key)
}

func LPopCount(ctx context.Context, key string, count int) *redis.StringSliceCmd {
	return rd.LPopCount(ctx, key, count)
}

func LPos(ctx context.Context, key string, value string, args redis.LPosArgs) *redis.IntCmd {
	return rd.LPos(ctx, key, value, args)
}

func LPosCount(ctx context.Context, key string, value string, count int64, args redis.LPosArgs) *redis.IntSliceCmd {
	return rd.LPosCount(ctx, key, value, count, args)
}

func LPush(ctx context.Context, key string, values ...interface{}) *redis.IntCmd {
	return rd.LPush(ctx, key, values...)
}

func LPushX(ctx context.Context, key string, values ...interface{}) *redis.IntCmd {
	return rd.LPushX(ctx, key, values...)
}

func LRange(ctx context.Context, key string, start, stop int64) *redis.StringSliceCmd {
	return rd.LRange(ctx, key, start, stop)
}

func LRem(ctx context.Context, key string, count int64, value interface{}) *redis.IntCmd {
	return rd.LRem(ctx, key, count, value)
}

func LSet(ctx context.Context, key string, index int64, value interface{}) *redis.StatusCmd {
	return rd.LSet(ctx, key, index, value)
}

func LTrim(ctx context.Context, key string, start, stop int64) *redis.StatusCmd {
	return rd.LTrim(ctx, key, start, stop)
}

func RPop(ctx context.Context, key string) *redis.StringCmd {
	return rd.RPop(ctx, key)
}

func RPopCount(ctx context.Context, key string, count int) *redis.StringSliceCmd {
	return rd.RPopCount(ctx, key, count)
}

func RPopLPush(ctx context.Context, source, destination string) *redis.StringCmd {
	return rd.RPopLPush(ctx, source, destination)
}

func RPush(ctx context.Context, key string, values ...interface{}) *redis.IntCmd {
	return rd.RPush(ctx, key, values...)
}

func RPushX(ctx context.Context, key string, values ...interface{}) *redis.IntCmd {
	return rd.RPushX(ctx, key, values...)
}

// Set 相关操作
func SAdd(ctx context.Context, key string, members ...interface{}) *redis.IntCmd {
	return rd.SAdd(ctx, key, members...)
}

func SCard(ctx context.Context, key string) *redis.IntCmd {
	return rd.SCard(ctx, key)
}

func SDiff(ctx context.Context, keys ...string) *redis.StringSliceCmd {
	return rd.SDiff(ctx, keys...)
}

func SDiffStore(ctx context.Context, destination string, keys ...string) *redis.IntCmd {
	return rd.SDiffStore(ctx, destination, keys...)
}

func SInter(ctx context.Context, keys ...string) *redis.StringSliceCmd {
	return rd.SInter(ctx, keys...)
}

func SInterStore(ctx context.Context, destination string, keys ...string) *redis.IntCmd {
	return rd.SInterStore(ctx, destination, keys...)
}

func SIsMember(ctx context.Context, key string, member interface{}) *redis.BoolCmd {
	return rd.SIsMember(ctx, key, member)
}

func SMembers(ctx context.Context, key string) *redis.StringSliceCmd {
	return rd.SMembers(ctx, key)
}

func SMembersMap(ctx context.Context, key string) *redis.StringStructMapCmd {
	return rd.SMembersMap(ctx, key)
}

func SMove(ctx context.Context, source, destination string, member interface{}) *redis.BoolCmd {
	return rd.SMove(ctx, source, destination, member)
}

func SPop(ctx context.Context, key string) *redis.StringCmd {
	return rd.SPop(ctx, key)
}

func SPopN(ctx context.Context, key string, count int64) *redis.StringSliceCmd {
	return rd.SPopN(ctx, key, count)
}

func SRandMember(ctx context.Context, key string) *redis.StringCmd {
	return rd.SRandMember(ctx, key)
}

func SRandMemberN(ctx context.Context, key string, count int64) *redis.StringSliceCmd {
	return rd.SRandMemberN(ctx, key, count)
}

func SRem(ctx context.Context, key string, members ...interface{}) *redis.IntCmd {
	return rd.SRem(ctx, key, members...)
}

func SUnion(ctx context.Context, keys ...string) *redis.StringSliceCmd {
	return rd.SUnion(ctx, keys...)
}

func SUnionStore(ctx context.Context, destination string, keys ...string) *redis.IntCmd {
	return rd.SUnionStore(ctx, destination, keys...)
}
func SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.BoolCmd {
	return rd.SetNX(ctx, key, value, expiration)
}

func SetRange(ctx context.Context, key string, offset int64, value string) *redis.IntCmd {
	return rd.SetRange(ctx, key, offset, value)
}

func StrLen(ctx context.Context, key string) *redis.IntCmd {
	return rd.StrLen(ctx, key)
}

// SortedSet 相关操作
func ZAdd(ctx context.Context, key string, members ...redis.Z) *redis.IntCmd {
	return rd.ZAdd(ctx, key, members...)
}

func ZAddNX(ctx context.Context, key string, members ...redis.Z) *redis.IntCmd {
	return rd.ZAddNX(ctx, key, members...)
}

func ZAddXX(ctx context.Context, key string, members ...redis.Z) *redis.IntCmd {
	return rd.ZAddXX(ctx, key, members...)
}

func ZCard(ctx context.Context, key string) *redis.IntCmd {
	return rd.ZCard(ctx, key)
}

func ZCount(ctx context.Context, key, min, max string) *redis.IntCmd {
	return rd.ZCount(ctx, key, min, max)
}

func ZIncrBy(ctx context.Context, key string, increment float64, member string) *redis.FloatCmd {
	return rd.ZIncrBy(ctx, key, increment, member)
}

func ZInterStore(ctx context.Context, destination string, store *redis.ZStore) *redis.IntCmd {
	return rd.ZInterStore(ctx, destination, store)
}

func ZLexCount(ctx context.Context, key, min, max string) *redis.IntCmd {
	return rd.ZLexCount(ctx, key, min, max)
}

func ZPopMax(ctx context.Context, key string, count ...int64) *redis.ZSliceCmd {
	return rd.ZPopMax(ctx, key, count...)
}

func ZPopMin(ctx context.Context, key string, count ...int64) *redis.ZSliceCmd {
	return rd.ZPopMin(ctx, key, count...)
}

func ZRange(ctx context.Context, key string, start, stop int64) *redis.StringSliceCmd {
	return rd.ZRange(ctx, key, start, stop)
}

func ZRangeByLex(ctx context.Context, key string, opt *redis.ZRangeBy) *redis.StringSliceCmd {
	return rd.ZRangeByLex(ctx, key, opt)
}

func ZRangeByScore(ctx context.Context, key string, opt *redis.ZRangeBy) *redis.StringSliceCmd {
	return rd.ZRangeByScore(ctx, key, opt)
}

func ZRank(ctx context.Context, key, member string) *redis.IntCmd {
	return rd.ZRank(ctx, key, member)
}

func ZRem(ctx context.Context, key string, members ...interface{}) *redis.IntCmd {
	return rd.ZRem(ctx, key, members...)
}

func ZRemRangeByLex(ctx context.Context, key, min, max string) *redis.IntCmd {
	return rd.ZRemRangeByLex(ctx, key, min, max)
}

func ZRemRangeByRank(ctx context.Context, key string, start, stop int64) *redis.IntCmd {
	return rd.ZRemRangeByRank(ctx, key, start, stop)
}

func ZRemRangeByScore(ctx context.Context, key, min, max string) *redis.IntCmd {
	return rd.ZRemRangeByScore(ctx, key, min, max)
}

func ZRevRange(ctx context.Context, key string, start, stop int64) *redis.StringSliceCmd {
	return rd.ZRevRange(ctx, key, start, stop)
}

func ZRevRangeByLex(ctx context.Context, key string, opt *redis.ZRangeBy) *redis.StringSliceCmd {
	return rd.ZRevRangeByLex(ctx, key, opt)
}

func ZRevRangeByScore(ctx context.Context, key string, opt *redis.ZRangeBy) *redis.StringSliceCmd {
	return rd.ZRevRangeByScore(ctx, key, opt)
}

func ZRevRank(ctx context.Context, key, member string) *redis.IntCmd {
	return rd.ZRevRank(ctx, key, member)
}

func ZScore(ctx context.Context, key, member string) *redis.FloatCmd {
	return rd.ZScore(ctx, key, member)
}

func ZUnionStore(ctx context.Context, dest string, store *redis.ZStore) *redis.IntCmd {
	return rd.ZUnionStore(ctx, dest, store)
}

// Stream 相关操作
func XAdd(ctx context.Context, a *redis.XAddArgs) *redis.StringCmd {
	return rd.XAdd(ctx, a)
}

func XDel(ctx context.Context, stream string, ids ...string) *redis.IntCmd {
	return rd.XDel(ctx, stream, ids...)
}

func XLen(ctx context.Context, stream string) *redis.IntCmd {
	return rd.XLen(ctx, stream)
}

func XRange(ctx context.Context, stream, start, stop string) *redis.XMessageSliceCmd {
	return rd.XRange(ctx, stream, start, stop)
}

func XRangeN(ctx context.Context, stream, start, stop string, count int64) *redis.XMessageSliceCmd {
	return rd.XRangeN(ctx, stream, start, stop, count)
}

func XRevRange(ctx context.Context, stream, start, stop string) *redis.XMessageSliceCmd {
	return rd.XRevRange(ctx, stream, start, stop)
}

func XRevRangeN(ctx context.Context, stream, start, stop string, count int64) *redis.XMessageSliceCmd {
	return rd.XRevRangeN(ctx, stream, start, stop, count)
}

func XRead(ctx context.Context, a *redis.XReadArgs) *redis.XStreamSliceCmd {
	return rd.XRead(ctx, a)
}

func XReadStreams(ctx context.Context, streams ...string) *redis.XStreamSliceCmd {
	return rd.XReadStreams(ctx, streams...)
}

func XGroupCreate(ctx context.Context, stream, group, start string) *redis.StatusCmd {
	return rd.XGroupCreate(ctx, stream, group, start)
}

func XGroupCreateMkStream(ctx context.Context, stream, group, start string) *redis.StatusCmd {
	return rd.XGroupCreateMkStream(ctx, stream, group, start)
}

func XGroupSetID(ctx context.Context, stream, group, start string) *redis.StatusCmd {
	return rd.XGroupSetID(ctx, stream, group, start)
}

func XGroupDestroy(ctx context.Context, stream, group string) *redis.IntCmd {
	return rd.XGroupDestroy(ctx, stream, group)
}

func XGroupCreateConsumer(ctx context.Context, stream, group, consumer string) *redis.IntCmd {
	return rd.XGroupCreateConsumer(ctx, stream, group, consumer)
}

func XGroupDelConsumer(ctx context.Context, stream, group, consumer string) *redis.IntCmd {
	return rd.XGroupDelConsumer(ctx, stream, group, consumer)
}

func XReadGroup(ctx context.Context, a *redis.XReadGroupArgs) *redis.XStreamSliceCmd {
	return rd.XReadGroup(ctx, a)
}

func XAck(ctx context.Context, stream, group string, ids ...string) *redis.IntCmd {
	return rd.XAck(ctx, stream, group, ids...)
}

func XPending(ctx context.Context, stream, group string) *redis.XPendingCmd {
	return rd.XPending(ctx, stream, group)
}

func XPendingExt(ctx context.Context, a *redis.XPendingExtArgs) *redis.XPendingExtCmd {
	return rd.XPendingExt(ctx, a)
}

func XClaim(ctx context.Context, a *redis.XClaimArgs) *redis.XMessageSliceCmd {
	return rd.XClaim(ctx, a)
}

func XClaimJustID(ctx context.Context, a *redis.XClaimArgs) *redis.StringSliceCmd {
	return rd.XClaimJustID(ctx, a)
}

func XAutoClaim(ctx context.Context, a *redis.XAutoClaimArgs) *redis.XAutoClaimCmd {
	return rd.XAutoClaim(ctx, a)
}

func XAutoClaimJustID(ctx context.Context, a *redis.XAutoClaimArgs) *redis.XAutoClaimJustIDCmd {
	return rd.XAutoClaimJustID(ctx, a)
}

func XTrimMaxLen(ctx context.Context, key string, maxLen int64) *redis.IntCmd {
	return rd.XTrimMaxLen(ctx, key, maxLen)
}

func XTrimMaxLenApprox(ctx context.Context, key string, maxLen, limit int64) *redis.IntCmd {
	return rd.XTrimMaxLenApprox(ctx, key, maxLen, limit)
}

func XTrimMinID(ctx context.Context, key string, minID string) *redis.IntCmd {
	return rd.XTrimMinID(ctx, key, minID)
}

func XTrimMinIDApprox(ctx context.Context, key string, minID string, limit int64) *redis.IntCmd {
	return rd.XTrimMinIDApprox(ctx, key, minID, limit)
}

func XInfoGroups(ctx context.Context, key string) *redis.XInfoGroupsCmd {
	return rd.XInfoGroups(ctx, key)
}

func XInfoStream(ctx context.Context, key string) *redis.XInfoStreamCmd {
	return rd.XInfoStream(ctx, key)
}

func XInfoStreamFull(ctx context.Context, key string, count int) *redis.XInfoStreamFullCmd {
	return rd.XInfoStreamFull(ctx, key, count)
}

func XInfoConsumers(ctx context.Context, key, group string) *redis.XInfoConsumersCmd {
	return rd.XInfoConsumers(ctx, key, group)
}

// 脚本相关操作
func Eval(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd {
	return rd.Eval(ctx, script, keys, args...)
}

func EvalSha(ctx context.Context, sha1 string, keys []string, args ...interface{}) *redis.Cmd {
	return rd.EvalSha(ctx, sha1, keys, args...)
}

func ScriptExists(ctx context.Context, scripts ...string) *redis.BoolSliceCmd {
	return rd.ScriptExists(ctx, scripts...)
}

func ScriptLoad(ctx context.Context, script string) *redis.StringCmd {
	return rd.ScriptLoad(ctx, script)
}

func ScriptFlush(ctx context.Context) *redis.StatusCmd {
	return rd.ScriptFlush(ctx)
}

func ScriptKill(ctx context.Context) *redis.StatusCmd {
	return rd.ScriptKill(ctx)
}

// 发布订阅相关操作
func Publish(ctx context.Context, channel string, message interface{}) *redis.IntCmd {
	return rd.Publish(ctx, channel, message)
}
func SPublish(ctx context.Context, channel string, message interface{}) *redis.IntCmd {
	return rd.SPublish(ctx, channel, message)
}

func PubSubChannels(ctx context.Context, pattern string) *redis.StringSliceCmd {
	return rd.PubSubChannels(ctx, pattern)
}

func PubSubNumSub(ctx context.Context, channels ...string) *redis.MapStringIntCmd {
	return rd.PubSubNumSub(ctx, channels...)
}

func PubSubNumPat(ctx context.Context) *redis.IntCmd {
	return rd.PubSubNumPat(ctx)
}

// 连接管理相关操作
func Ping(ctx context.Context) *redis.StatusCmd {
	return rd.Ping(ctx)
}

func ClientID(ctx context.Context) *redis.IntCmd {
	return rd.ClientID(ctx)
}

func ClientInfo(ctx context.Context) *redis.ClientInfoCmd {
	return rd.ClientInfo(ctx)
}

func ClientList(ctx context.Context) *redis.StringCmd {
	return rd.ClientList(ctx)
}

func ClientKill(ctx context.Context, ipPort string) *redis.StatusCmd {
	return rd.ClientKill(ctx, ipPort)
}

func ClientKillByFilter(ctx context.Context, keys ...string) *redis.IntCmd {
	return rd.ClientKillByFilter(ctx, keys...)
}

func ClientPause(ctx context.Context, dur time.Duration) *redis.BoolCmd {
	return rd.ClientPause(ctx, dur)
}

func ClientUnblock(ctx context.Context, id int64) *redis.IntCmd {
	return rd.ClientUnblock(ctx, id)
}

func ClientUnblockWithError(ctx context.Context, id int64) *redis.IntCmd {
	return rd.ClientUnblockWithError(ctx, id)
}

// 服务器相关操作
func Time(ctx context.Context) *redis.TimeCmd {
	return rd.Time(ctx)
}

func DBSize(ctx context.Context) *redis.IntCmd {
	return rd.DBSize(ctx)
}

func FlushAll(ctx context.Context) *redis.StatusCmd {
	return rd.FlushAll(ctx)
}

func FlushAllAsync(ctx context.Context) *redis.StatusCmd {
	return rd.FlushAllAsync(ctx)
}

func FlushDB(ctx context.Context) *redis.StatusCmd {
	return rd.FlushDB(ctx)
}

func FlushDBAsync(ctx context.Context) *redis.StatusCmd {
	return rd.FlushDBAsync(ctx)
}

func Info(ctx context.Context, section ...string) *redis.StringCmd {
	return rd.Info(ctx, section...)
}

func Save(ctx context.Context) *redis.StatusCmd {
	return rd.Save(ctx)
}

func Shutdown(ctx context.Context) *redis.StatusCmd {
	return rd.Shutdown(ctx)
}

func ShutdownSave(ctx context.Context) *redis.StatusCmd {
	return rd.ShutdownSave(ctx)
}

func ShutdownNoSave(ctx context.Context) *redis.StatusCmd {
	return rd.ShutdownNoSave(ctx)
}

func SlaveOf(ctx context.Context, host, port string) *redis.StatusCmd {
	return rd.SlaveOf(ctx, host, port)
}

func SlowLogGet(ctx context.Context, num int64) *redis.SlowLogCmd {
	return rd.SlowLogGet(ctx, num)
}

func DebugObject(ctx context.Context, key string) *redis.StringCmd {
	return rd.DebugObject(ctx, key)
}

func MemoryUsage(ctx context.Context, key string, samples ...int) *redis.IntCmd {
	return rd.MemoryUsage(ctx, key, samples...)
}

// 位图(Bitmap)相关操作
func BitCount(ctx context.Context, key string, bitCount *redis.BitCount) *redis.IntCmd {
	return rd.BitCount(ctx, key, bitCount)
}

func BitOpAnd(ctx context.Context, destKey string, keys ...string) *redis.IntCmd {
	return rd.BitOpAnd(ctx, destKey, keys...)
}

func BitOpOr(ctx context.Context, destKey string, keys ...string) *redis.IntCmd {
	return rd.BitOpOr(ctx, destKey, keys...)
}

func BitOpXor(ctx context.Context, destKey string, keys ...string) *redis.IntCmd {
	return rd.BitOpXor(ctx, destKey, keys...)
}

func BitOpNot(ctx context.Context, destKey string, key string) *redis.IntCmd {
	return rd.BitOpNot(ctx, destKey, key)
}

func BitPos(ctx context.Context, key string, bit int64, pos ...int64) *redis.IntCmd {
	return rd.BitPos(ctx, key, bit, pos...)
}

func BitField(ctx context.Context, key string, args ...interface{}) *redis.IntSliceCmd {
	return rd.BitField(ctx, key, args...)
}

func GetBit(ctx context.Context, key string, offset int64) *redis.IntCmd {
	return rd.GetBit(ctx, key, offset)
}

func SetBit(ctx context.Context, key string, offset int64, value int) *redis.IntCmd {
	return rd.SetBit(ctx, key, offset, value)
}

func HDel(ctx context.Context, key string, fields ...string) *redis.IntCmd {
	return rd.HDel(ctx, key, fields...)
}
func HExists(ctx context.Context, key, field string) *redis.BoolCmd {
	return rd.HExists(ctx, key, field)
}
func HGet(ctx context.Context, key, field string) *redis.StringCmd {
	return rd.HGet(ctx, key, field)
}
func HGetAll(ctx context.Context, key string) *redis.MapStringStringCmd {
	return rd.HGetAll(ctx, key)
}

func HIncrBy(ctx context.Context, key, field string, incr int64) *redis.IntCmd {
	return rd.HIncrBy(ctx, key, field, incr)
}
func HIncrByFloat(ctx context.Context, key, field string, incr float64) *redis.FloatCmd {
	return rd.HIncrByFloat(ctx, key, field, incr)
}

func HKeys(ctx context.Context, key string) *redis.StringSliceCmd {
	return rd.HKeys(ctx, key)
}
func HLen(ctx context.Context, key string) *redis.IntCmd {
	return rd.HLen(ctx, key)
}
func HMGet(ctx context.Context, key string, fields ...string) *redis.SliceCmd {
	return rd.HMGet(ctx, key, fields...)
}
func HSet(ctx context.Context, key string, values ...interface{}) *redis.IntCmd {
	return rd.HSet(ctx, key, values...)
}

func HSetNX(ctx context.Context, key, field string, value interface{}) *redis.BoolCmd {
	return rd.HSetNX(ctx, key, field, value)
}
func HScan(ctx context.Context, key string, cursor uint64, match string, count int64) *redis.ScanCmd {
	return rd.HScan(ctx, key, cursor, match, count)
}
func HScanNoValues(ctx context.Context, key string, cursor uint64, match string, count int64) *redis.ScanCmd {
	return rd.HScanNoValues(ctx, key, cursor, match, count)
}
func HVals(ctx context.Context, key string) *redis.StringSliceCmd {
	return rd.HVals(ctx, key)
}

func HRandField(ctx context.Context, key string, count int) *redis.StringSliceCmd {
	return rd.HRandField(ctx, key, count)
}
func HRandFieldWithValues(ctx context.Context, key string, count int) *redis.KeyValueSliceCmd {
	return rd.HRandFieldWithValues(ctx, key, count)
}
func HExpire(ctx context.Context, key string, expiration time.Duration, fields ...string) *redis.IntSliceCmd {
	return rd.HExpire(ctx, key, expiration, fields...)
}
func HExpireWithArgs(ctx context.Context, key string, expiration time.Duration, expirationArgs redis.HExpireArgs, fields ...string) *redis.IntSliceCmd {
	return rd.HExpireWithArgs(ctx, key, expiration, expirationArgs, fields...)
}
func HPExpire(ctx context.Context, key string, expiration time.Duration, fields ...string) *redis.IntSliceCmd {
	return rd.HPExpire(ctx, key, expiration, fields...)
}
func HPExpireWithArgs(ctx context.Context, key string, expiration time.Duration, expirationArgs redis.HExpireArgs, fields ...string) *redis.IntSliceCmd {
	return rd.HPExpireWithArgs(ctx, key, expiration, expirationArgs, fields...)
}
func HExpireAt(ctx context.Context, key string, tm time.Time, fields ...string) *redis.IntSliceCmd {
	return rd.HExpireAt(ctx, key, tm, fields...)
}
func HExpireAtWithArgs(ctx context.Context, key string, tm time.Time, expirationArgs redis.HExpireArgs, fields ...string) *redis.IntSliceCmd {
	return rd.HExpireAtWithArgs(ctx, key, tm, expirationArgs, fields...)
}
func HPExpireAt(ctx context.Context, key string, tm time.Time, fields ...string) *redis.IntSliceCmd {
	return rd.HPExpireAt(ctx, key, tm, fields...)
}
func HPExpireAtWithArgs(ctx context.Context, key string, tm time.Time, expirationArgs redis.HExpireArgs, fields ...string) *redis.IntSliceCmd {
	return rd.HPExpireAtWithArgs(ctx, key, tm, expirationArgs, fields...)
}
func HPersist(ctx context.Context, key string, fields ...string) *redis.IntSliceCmd {
	return rd.HPersist(ctx, key, fields...)
}
func HExpireTime(ctx context.Context, key string, fields ...string) *redis.IntSliceCmd {
	return rd.HExpireTime(ctx, key, fields...)
}
func HPExpireTime(ctx context.Context, key string, fields ...string) *redis.IntSliceCmd {
	return rd.HPExpireTime(ctx, key, fields...)
}
func HTTL(ctx context.Context, key string, fields ...string) *redis.IntSliceCmd {
	return rd.HTTL(ctx, key, fields...)
}
func HPTTL(ctx context.Context, key string, fields ...string) *redis.IntSliceCmd {
	return rd.HPTTL(ctx, key, fields...)
}
func PFAdd(ctx context.Context, key string, els ...interface{}) *redis.IntCmd {
	return rd.PFAdd(ctx, key, els...)
}
func PFCount(ctx context.Context, keys ...string) *redis.IntCmd {
	return rd.PFCount(ctx, keys...)
}
func PFMerge(ctx context.Context, dest string, keys ...string) *redis.StatusCmd {
	return rd.PFMerge(ctx, dest, keys...)
}

func GeoAdd(ctx context.Context, key string, geoLocation ...*redis.GeoLocation) *redis.IntCmd {
	return rd.GeoAdd(ctx, key, geoLocation...)
}
func GeoPos(ctx context.Context, key string, members ...string) *redis.GeoPosCmd {
	return rd.GeoPos(ctx, key, members...)
}
func GeoRadius(ctx context.Context, key string, longitude, latitude float64, query *redis.GeoRadiusQuery) *redis.GeoLocationCmd {
	return rd.GeoRadius(ctx, key, longitude, latitude, query)
}
func GeoRadiusStore(ctx context.Context, key string, longitude, latitude float64, query *redis.GeoRadiusQuery) *redis.IntCmd {
	return rd.GeoRadiusStore(ctx, key, longitude, latitude, query)
}
func GeoRadiusByMember(ctx context.Context, key, member string, query *redis.GeoRadiusQuery) *redis.GeoLocationCmd {
	return rd.GeoRadiusByMember(ctx, key, member, query)
}
func GeoRadiusByMemberStore(ctx context.Context, key, member string, query *redis.GeoRadiusQuery) *redis.IntCmd {
	return rd.GeoRadiusByMemberStore(ctx, key, member, query)
}
func GeoSearch(ctx context.Context, key string, q *redis.GeoSearchQuery) *redis.StringSliceCmd {
	return rd.GeoSearch(ctx, key, q)
}
func GeoSearchLocation(ctx context.Context, key string, q *redis.GeoSearchLocationQuery) *redis.GeoSearchLocationCmd {
	return rd.GeoSearchLocation(ctx, key, q)
}
func GeoSearchStore(ctx context.Context, key, store string, q *redis.GeoSearchStoreQuery) *redis.IntCmd {
	return rd.GeoSearchStore(ctx, key, store, q)
}
func GeoDist(ctx context.Context, key string, member1, member2, unit string) *redis.FloatCmd {
	return rd.GeoDist(ctx, key, member1, member2, unit)
}
func GeoHash(ctx context.Context, key string, members ...string) *redis.StringSliceCmd {
	return rd.GeoHash(ctx, key, members...)
}

// 添加连接池统计信息获取
func PoolStats() *redis.PoolStats {
	if cluster, ok := rd.ClientInf.(*redis.ClusterClient); ok {
		return cluster.PoolStats()
	}
	if client, ok := rd.ClientInf.(*redis.Client); ok {
		return client.PoolStats()
	}
	return nil
}

// 添加健康检查
func PingWithTimeout(ctx context.Context, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return rd.Ping(ctx).Err()
}
