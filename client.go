package redis

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis/v7"
)

//Config 配置
type Config struct {
	Type      bool     //是否集群
	Hosts     []string //IP
	Password  string   //密码
	Database  int      //数据库
	PoolSize  int      //连接池大小
	KeyPrefix string
}

//Configs 配置组
type Configs struct {
	cfg         map[string]*Config
	connections map[string]*Client
	mu          sync.RWMutex
}

//Default 构造
func Default() *Configs {
	return &Configs{
		cfg:         make(map[string]*Config),
		connections: make(map[string]*Client),
	}
}

//SetConfig 设置配置文件
func (configs *Configs) SetConfig(name string, cf *Config) *Configs {
	configs.cfg[name] = cf
	return configs
}

//GetRedis  获取 redis 实列
func (configs *Configs) GetRedis(name string) *Client {
	conn, ok := configs.connections[name]
	if ok {
		return conn
	}
	config, ok := configs.cfg[name]
	if !ok {
		Log.Panic("Redis配置:" + name + "找不到！")
	}

	db := connect(config)
	configs.mu.Lock()
	configs.connections[name] = db
	configs.mu.Unlock()
	configs.mu.RLock()
	v := configs.connections[name]
	configs.mu.RUnlock()
	return v
}

func connect(config *Config) *Client {
	opts := Options{}
	if config.Type {
		opts.Type = ClientCluster
	} else {
		opts.Type = ClientNormal
	}
	opts.Hosts = config.Hosts
	opts.KeyPrefix = config.KeyPrefix

	if config.PoolSize > 0 {
		opts.PoolSize = config.PoolSize
	} else {
		opts.PoolSize = 64
	}
	if config.Database > 0 {
		opts.Database = config.Database
	} else {
		opts.Database = 0
	}
	if config.Password != "" {
		opts.Password = config.Password
	}
	client := NewClient(opts)
	if err := client.Ping().Err(); err != nil {
		Log.Panic(err.Error())
	}
	return client
}

// RedisNil means nil reply, .e.g. when key does not exist.
const RedisNil = redis.Nil

// Client a struct representing the redis client
type Client struct {
	opts      Options
	client    Commander
	fmtString string
}

// NewClient 新客户端
func NewClient(opts Options) *Client {
	r := &Client{opts: opts}

	switch opts.Type {
	// 群集客户端
	case ClientCluster:
		tc := redis.NewClusterClient(opts.GetClusterConfig())
		// redisClient.AddHook()
		r.client = tc
	// 标准客户端也是默认值
	case ClientNormal:
		fallthrough
	default:
		tc := redis.NewClient(opts.GetNormalConfig())
		// redisClient.AddHook()
		r.client = tc

	}
	r.fmtString = opts.KeyPrefix + "%s"
	return r
}

// IsCluster 判断是否集群
func (r *Client) IsCluster() bool {
	if r.opts.Type == ClientCluster {
		return true
	}
	return false
}

//Prefix 返回前缀+键
func (r *Client) Prefix(key string) string {
	return fmt.Sprintf(r.fmtString, key)
}

// k 格式化并返回带前缀的密钥
func (r *Client) k(key string) string {
	return fmt.Sprintf(r.fmtString, key)
}

// ks 使用前缀格式化并返回一组键
func (r *Client) ks(key ...string) []string {
	keys := make([]string, len(key))
	for i, k := range key {
		keys[i] = r.k(k)
	}
	return keys
}

// GetClient 返回客户端
func (r *Client) GetClient() Commander {
	return r.client
}

// Ping 使用客户端向 Redis 服务器发送一个 PING ，如果服务器运作正常的话，会返回一个 PONG 。
// 通常用于测试与服务器的连接是否仍然生效，或者用于测量延迟值。
// 如果连接正常就返回一个 PONG ，否则返回一个连接错误。
func (r *Client) Ping() *redis.StatusCmd {
	return r.client.Ping()
}

// Incr 将 key 中储存的数字值增一。
// 如果 key 不存在，那么 key 的值会先被初始化为 0 ，然后再执行 INCR 操作。
// 如果值包含错误的类型，或字符串类型的值不能表示为数字，那么返回一个错误。
// 本操作的值限制在 64 位(bit)有符号数字表示之内。
// 执行 INCR 命令之后 key 的值。
func (r *Client) Incr(key string) *redis.IntCmd {
	return r.client.Incr(r.k(key))
}

// IncrBy 将 key 所储存的值加上增量 increment 。
// 如果 key 不存在，那么 key 的值会先被初始化为 0 ，然后再执行 INCRBY 命令。
// 如果值包含错误的类型，或字符串类型的值不能表示为数字，那么返回一个错误。
// 本操作的值限制在 64 位(bit)有符号数字表示之内。
// 关于递增(increment) / 递减(decrement)操作的更多信息，参见 INCR 命令。
// 加上 increment 之后， key 的值。
func (r *Client) IncrBy(key string, value int64) *redis.IntCmd {
	return r.client.IncrBy(r.k(key), value)
}

// Decr 将 key 中储存的数字值减一。
// 如果 key 不存在，那么 key 的值会先被初始化为 0 ，然后再执行 DECR 操作。
// 如果值包含错误的类型，或字符串类型的值不能表示为数字，那么返回一个错误。
// 本操作的值限制在 64 位(bit)有符号数字表示之内。
// 关于递增(increment) / 递减(decrement)操作的更多信息，请参见 INCR 命令。
// 执行 DECR 命令之后 key 的值。
func (r *Client) Decr(key string) *redis.IntCmd {
	return r.client.Decr(r.k(key))
}

// DecrBy 将 key 所储存的值减去减量 decrement 。
// 如果 key 不存在，那么 key 的值会先被初始化为 0 ，然后再执行 DECRBY 操作。
// 如果值包含错误的类型，或字符串类型的值不能表示为数字，那么返回一个错误。
// 本操作的值限制在 64 位(bit)有符号数字表示之内。
// 关于更多递增(increment) / 递减(decrement)操作的更多信息，请参见 INCR 命令。
// 减去 decrement 之后， key 的值。
func (r *Client) DecrBy(key string, value int64) *redis.IntCmd {
	return r.client.DecrBy(r.k(key), value)
}

// Expire 为给定 key 设置生存时间，当 key 过期时(生存时间为 0 )，它会被自动删除。
// 设置成功返回 1 。
// 当 key 不存在或者不能为 key 设置生存时间时(比如在低于 2.1.3 版本的 Redis 中你尝试更新 key 的生存时间)，返回 0 。
func (r *Client) Expire(key string, expiration time.Duration) *redis.BoolCmd {
	return r.client.Expire(r.k(key), expiration)
}

// ExpireAt  EXPIREAT 的作用和 EXPIRE 类似，都用于为 key 设置生存时间。
// 命令用于以 UNIX 时间戳(unix timestamp)格式设置 key 的过期时间
func (r *Client) ExpireAt(key string, tm time.Time) *redis.BoolCmd {
	return r.client.ExpireAt(r.k(key), tm)
}

// Persist 移除给定 key 的生存时间，将这个 key 从『易失的』(带生存时间 key )转换成『持久的』(一个不带生存时间、永不过期的 key )。
// 当生存时间移除成功时，返回 1 .
// 如果 key 不存在或 key 没有设置生存时间，返回 0 。
func (r *Client) Persist(key string) *redis.BoolCmd {
	return r.client.Persist(r.k(key))
}

// PExpire 毫秒为单位设置 key 的生存时间
func (r *Client) PExpire(key string, expiration time.Duration) *redis.BoolCmd {
	return r.client.PExpire(r.k(key), expiration)
}

// PExpireAt 这个命令和 expireat 命令类似，但它以毫秒为单位设置 key 的过期 unix 时间戳，而不是像 expireat 那样，以秒为单位。
// 如果生存时间设置成功，返回 1 。 当 key 不存在或没办法设置生存时间时，返回 0
func (r *Client) PExpireAt(key string, tm time.Time) *redis.BoolCmd {
	return r.client.PExpireAt(r.k(key), tm)
}

// PTTL 这个命令类似于 TTL 命令，但它以毫秒为单位返回 key 的剩余生存时间，而不是像 TTL 命令那样，以秒为单位。
// 当 key 不存在时，返回 -2 。
// 当 key 存在但没有设置剩余生存时间时，返回 -1 。
// 否则，以毫秒为单位，返回 key 的剩余生存时间。
func (r *Client) PTTL(key string) *redis.DurationCmd {
	return r.client.PTTL(r.k(key))
}

// TTL 以秒为单位，返回给定 key 的剩余生存时间(TTL, time to live)。
// 当 key 不存在时，返回 -2 。
// 当 key 存在但没有设置剩余生存时间时，返回 -1 。
// 否则，以秒为单位，返回 key 的剩余生存时间。
func (r *Client) TTL(key string) *redis.DurationCmd {
	return r.client.TTL(r.k(key))
}

// Exists 检查给定 key 是否存在。
// 若 key 存在，返回 1 ，否则返回 0 。
func (r *Client) Exists(key ...string) *redis.IntCmd {
	return r.client.Exists(r.ks(key...)...)
}

// Get 返回 key 所关联的字符串值。
// 如果 key 不存在那么返回特殊值 nil 。
// 假如 key 储存的值不是字符串类型，返回一个错误，因为 GET 只能用于处理字符串值。
// 当 key 不存在时，返回 nil ，否则，返回 key 的值。
// 如果 key 不是字符串类型，那么返回一个错误。
func (r *Client) Get(key string) *redis.StringCmd {
	return r.client.Get(r.k(key))
}

// GetBit 对 key 所储存的字符串值，获取指定偏移量上的位(bit)。
// 当 offset 比字符串值的长度大，或者 key 不存在时，返回 0 。
// 字符串值指定偏移量上的位(bit)。
func (r *Client) GetBit(key string, offset int64) *redis.IntCmd {
	return r.client.GetBit(r.k(key), offset)
}

// GetRange 返回 key 中字符串值的子字符串，字符串的截取范围由 start 和 end 两个偏移量决定(包括 start 和 end 在内)。
// 负数偏移量表示从字符串最后开始计数， -1 表示最后一个字符， -2 表示倒数第二个，以此类推。
// GETRANGE 通过保证子字符串的值域(range)不超过实际字符串的值域来处理超出范围的值域请求。
// 返回截取得出的子字符串。
func (r *Client) GetRange(key string, start, end int64) *redis.StringCmd {
	return r.client.GetRange(r.k(key), start, end)
}

// GetSet 将给定 key 的值设为 value ，并返回 key 的旧值(old value)。
// 当 key 存在但不是字符串类型时，返回一个错误。
// 返回给定 key 的旧值。
// 当 key 没有旧值时，也即是， key 不存在时，返回 nil 。
func (r *Client) GetSet(key string, value interface{}) *redis.StringCmd {
	return r.client.GetSet(r.k(key), value)
}

// MGetByPipeline gets multiple values from keys,Pipeline is used when
// redis is a cluster,This means higher IO performance
// params: keys ...string
// return: []string, error
func (r *Client) MGetByPipeline(keys ...string) ([]string, error) {

	var res []string

	if r.IsCluster() {
		start := time.Now()
		pipeLineLen := 100
		pipeCount := len(keys)/pipeLineLen + 1
		pipes := make([]redis.Pipeliner, pipeCount)
		for i := 0; i < pipeCount; i++ {
			pipes[i] = r.client.Pipeline()
		}
		for i, k := range keys {
			p := pipes[i%pipeCount]
			p.Get(r.k(k))
		}
		Log.Debug("process cost: %v", time.Since(start))
		start = time.Now()
		var wg sync.WaitGroup
		var lock sync.Mutex
		errors := make(chan error, pipeCount)
		for _, p := range pipes {
			p := p
			wg.Add(1)
			go func() {
				defer wg.Done()
				cmders, err := p.Exec()
				if err != nil {
					select {
					case errors <- err:
					default:
					}
					return
				}
				lock.Lock()
				defer lock.Unlock()
				for _, cmder := range cmders {
					result, _ := cmder.(*redis.StringCmd).Result()
					res = append(res, result)
				}
			}()
		}
		wg.Wait()
		Log.Debug("exec cost: %v", time.Since(start))

		if len(errors) > 0 {
			return nil, <-errors
		}

		return res, nil
	}

	vals, err := r.client.MGet(keys...).Result()

	if redis.Nil != err && nil != err {
		return nil, err
	}

	for _, item := range vals {
		res = append(res, fmt.Sprintf("%s", item))
	}

	return res, err
}

// MGet 返回所有(一个或多个)给定 key 的值。
// 如果给定的 key 里面，有某个 key 不存在，那么这个 key 返回特殊值 nil 。因此，该命令永不失败。
// 一个包含所有给定 key 的值的列表。
func (r *Client) MGet(keys ...string) *redis.SliceCmd {
	return r.client.MGet(r.ks(keys...)...)
}

// Dump 序列化给定 key ，并返回被序列化的值，使用 RESTORE 命令可以将这个值反序列化为 Redis 键。
// 如果 key 不存在，那么返回 nil 。
// 否则，返回序列化之后的值。
func (r *Client) Dump(key string) *redis.StringCmd {
	return r.client.Dump(r.k(key))
}

//HExists 查看哈希表 key 中，给定域 field 是否存在。
func (r *Client) HExists(key, field string) *redis.BoolCmd {
	return r.client.HExists(r.k(key), field)
}

// HGet 返回哈希表 key 中给定域 field 的值。
func (r *Client) HGet(key, field string) *redis.StringCmd {
	return r.client.HGet(r.k(key), field)
}

// HGetAll 返回哈希表 key 中，所有的域和值。
// 在返回值里，紧跟每个域名(field name)之后是域的值(value)，所以返回值的长度是哈希表大小的两倍。
func (r *Client) HGetAll(key string) *redis.StringStringMapCmd {
	return r.client.HGetAll(r.k(key))
}

// HIncrBy 为哈希表 key 中的域 field 的值加上增量 increment 。
// 增量也可以为负数，相当于对给定域进行减法操作。
// 如果 key 不存在，一个新的哈希表被创建并执行 HINCRBY 命令。
// 如果域 field 不存在，那么在执行命令前，域的值被初始化为 0 。
// 对一个储存字符串值的域 field 执行 HINCRBY 命令将造成一个错误。
// 本操作的值被限制在 64 位(bit)有符号数字表示之内。
func (r *Client) HIncrBy(key, field string, incr int64) *redis.IntCmd {
	return r.client.HIncrBy(r.k(key), field, incr)
}

// HIncrByFloat 为哈希表 key 中的域 field 加上浮点数增量 increment 。
// 如果哈希表中没有域 field ，那么 HINCRBYFLOAT 会先将域 field 的值设为 0 ，然后再执行加法操作。
// 如果键 key 不存在，那么 HINCRBYFLOAT 会先创建一个哈希表，再创建域 field ，最后再执行加法操作。
func (r *Client) HIncrByFloat(key, field string, incr float64) *redis.FloatCmd {
	return r.client.HIncrByFloat(r.k(key), field, incr)
}

// HKeys 返回哈希表 key 中的所有域。
func (r *Client) HKeys(key string) *redis.StringSliceCmd {
	return r.client.HKeys(r.k(key))
}

//HLen 返回哈希表 key 中域的数量。
func (r *Client) HLen(key string) *redis.IntCmd {
	return r.client.HLen(r.k(key))
}

// HMGet 返回哈希表 key 中，一个或多个给定域的值。
// 如果给定的域不存在于哈希表，那么返回一个 nil 值。
// 因为不存在的 key 被当作一个空哈希表来处理，所以对一个不存在的 key 进行 HMGET 操作将返回一个只带有 nil 值的表。
func (r *Client) HMGet(key string, fields ...string) *redis.SliceCmd {
	return r.client.HMGet(r.k(key), fields...)
}

// HMSet 同时将多个 field-value (域-值)对设置到哈希表 key 中。
// 此命令会覆盖哈希表中已存在的域。
// 如果 key 不存在，一个空哈希表被创建并执行 HMSET 操作。
func (r *Client) HMSet(key string, value ...interface{}) *redis.BoolCmd {
	return r.client.HMSet(r.k(key), value...)
}

// HSet 将哈希表 key 中的域 field 的值设为 value 。
// 如果 key 不存在，一个新的哈希表被创建并进行 HSET 操作。
// 如果域 field 已经存在于哈希表中，旧值将被覆盖。
func (r *Client) HSet(key string, value ...interface{}) *redis.IntCmd {
	return r.client.HSet(r.k(key), value...)
}

// HSetNX 将哈希表 key 中的域 field 的值设置为 value ，当且仅当域 field 不存在。
// 若域 field 已经存在，该操作无效。
// 如果 key 不存在，一个新哈希表被创建并执行 HSETNX 命令。
func (r *Client) HSetNX(key, field string, value interface{}) *redis.BoolCmd {
	return r.client.HSetNX(r.k(key), field, value)
}

// HVals 返回哈希表 key 中所有域的值。
func (r *Client) HVals(key string) *redis.StringSliceCmd {
	return r.client.HVals(r.k(key))
}

// HDel 删除哈希表 key 中的一个或多个指定域，不存在的域将被忽略。
func (r *Client) HDel(key string, fields ...string) *redis.IntCmd {
	return r.client.HDel(r.k(key), fields...)
}

// LIndex 返回列表 key 中，下标为 index 的元素。
// 下标(index)参数 start 和 stop 都以 0 为底，也就是说，以 0 表示列表的第一个元素，以 1 表示列表的第二个元素，以此类推。
// 你也可以使用负数下标，以 -1 表示列表的最后一个元素， -2 表示列表的倒数第二个元素，以此类推。
// 如果 key 不是列表类型，返回一个错误。
func (r *Client) LIndex(key string, index int64) *redis.StringCmd {
	return r.client.LIndex(r.k(key), index)
}

// LInsert 将值 value 插入到列表 key 当中，位于值 pivot 之前或之后。
// 当 pivot 不存在于列表 key 时，不执行任何操作。
// 当 key 不存在时， key 被视为空列表，不执行任何操作。
// 如果 key 不是列表类型，返回一个错误。
func (r *Client) LInsert(key, op string, pivot, value interface{}) *redis.IntCmd {
	return r.client.LInsert(r.k(key), op, pivot, value)
}

// LInsertAfter 同 LInsert
func (r *Client) LInsertAfter(key string, pivot, value interface{}) *redis.IntCmd {
	return r.client.LInsertAfter(r.k(key), pivot, value)
}

// LInsertBefore 同 LInsert
func (r *Client) LInsertBefore(key string, pivot, value interface{}) *redis.IntCmd {
	return r.client.LInsertBefore(r.k(key), pivot, value)
}

// LLen 返回列表 key 的长度。
// 如果 key 不存在，则 key 被解释为一个空列表，返回 0 .
// 如果 key 不是列表类型，返回一个错误。
func (r *Client) LLen(key string) *redis.IntCmd {
	return r.client.LLen(r.k(key))
}

// LPop 移除并返回列表 key 的头元素。
func (r *Client) LPop(key string) *redis.StringCmd {
	return r.client.LPop(r.k(key))
}

// LPush 将一个或多个值 value 插入到列表 key 的表头
// 如果有多个 value 值，那么各个 value 值按从左到右的顺序依次插入到表头
// 如果 key 不存在，一个空列表会被创建并执行 LPush 操作。
// 当 key 存在但不是列表类型时，返回一个错误。
func (r *Client) LPush(key string, values ...interface{}) *redis.IntCmd {
	return r.client.LPush(r.k(key), values...)
}

// LPushX 将值 value 插入到列表 key 的表头，当且仅当 key 存在并且是一个列表。
// 和 LPUSH 命令相反，当 key 不存在时， LPUSHX 命令什么也不做。
func (r *Client) LPushX(key string, value interface{}) *redis.IntCmd {
	return r.client.LPushX(r.k(key), value)
}

// LRange 返回列表 key 中指定区间内的元素，区间以偏移量 start 和 stop 指定。
// 下标(index)参数 start 和 stop 都以 0 为底，也就是说，以 0 表示列表的第一个元素，以 1 表示列表的第二个元素，以此类推。
// 你也可以使用负数下标，以 -1 表示列表的最后一个元素， -2 表示列表的倒数第二个元素，以此类推。
func (r *Client) LRange(key string, start, stop int64) *redis.StringSliceCmd {
	return r.client.LRange(r.k(key), start, stop)
}

// LRem 根据参数 count 的值，移除列表中与参数 value 相等的元素。
func (r *Client) LRem(key string, count int64, value interface{}) *redis.IntCmd {
	return r.client.LRem(r.k(key), count, value)
}

// LSet 将列表 key 下标为 index 的元素的值设置为 value 。
// 当 index 参数超出范围，或对一个空列表( key 不存在)进行 LSET 时，返回一个错误。
// 关于列表下标的更多信息，请参考 LINDEX 命令。
func (r *Client) LSet(key string, index int64, value interface{}) *redis.StatusCmd {
	return r.client.LSet(r.k(key), index, value)
}

// LTrim 对一个列表进行修剪(trim)，就是说，让列表只保留指定区间内的元素，不在指定区间之内的元素都将被删除。
// 举个例子，执行命令 LTRIM list 0 2 ，表示只保留列表 list 的前三个元素，其余元素全部删除。
// 下标(index)参数 start 和 stop 都以 0 为底，也就是说，以 0 表示列表的第一个元素，以 1 表示列表的第二个元素，以此类推。
// 你也可以使用负数下标，以 -1 表示列表的最后一个元素， -2 表示列表的倒数第二个元素，以此类推。
// 当 key 不是列表类型时，返回一个错误。
func (r *Client) LTrim(key string, start, stop int64) *redis.StatusCmd {
	return r.client.LTrim(r.k(key), start, stop)
}

// RPop 移除并返回列表 key 的尾元素。
func (r *Client) RPop(key string) *redis.StringCmd {
	return r.client.RPop(r.k(key))
}

// RPopLPush 命令 RPOPLPUSH 在一个原子时间内，执行以下两个动作：
// 将列表 source 中的最后一个元素(尾元素)弹出，并返回给客户端。
// 将 source 弹出的元素插入到列表 destination ，作为 destination 列表的的头元素。
// 举个例子，你有两个列表 source 和 destination ， source 列表有元素 a, b, c ， destination 列表有元素 x, y, z ，执行 RPOPLPUSH source destination 之后， source 列表包含元素 a, b ， destination 列表包含元素 c, x, y, z ，并且元素 c 会被返回给客户端。
// 如果 source 不存在，值 nil 被返回，并且不执行其他动作。
// 如果 source 和 destination 相同，则列表中的表尾元素被移动到表头，并返回该元素，可以把这种特殊情况视作列表的旋转(rotation)操作。
func (r *Client) RPopLPush(source, destination string) *redis.StringCmd {
	return r.client.RPopLPush(r.k(source), r.k(destination))
}

// RPush 将一个或多个值 value 插入到列表 key 的表尾(最右边)。
// 如果有多个 value 值，那么各个 value 值按从左到右的顺序依次插入到表尾：比如对一个空列表 mylist 执行 RPUSH mylist a b c ，得出的结果列表为 a b c ，等同于执行命令 RPUSH mylist a 、 RPUSH mylist b 、 RPUSH mylist c 。
// 如果 key 不存在，一个空列表会被创建并执行 RPUSH 操作。
// 当 key 存在但不是列表类型时，返回一个错误。
func (r *Client) RPush(key string, values ...interface{}) *redis.IntCmd {
	return r.client.RPush(r.k(key), values...)
}

// RPushX 将值 value 插入到列表 key 的表尾，当且仅当 key 存在并且是一个列表。
// 和 RPUSH 命令相反，当 key 不存在时， RPUSHX 命令什么也不做。
func (r *Client) RPushX(key string, value interface{}) *redis.IntCmd {
	return r.client.RPushX(r.k(key), value)
}

// Set 将字符串值 value 关联到 key 。
// 如果 key 已经持有其他值， SET 就覆写旧值，无视类型。
// 对于某个原本带有生存时间（TTL）的键来说， 当 SET 命令成功在这个键上执行时， 这个键原有的 TTL 将被清除。
func (r *Client) Set(key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	return r.client.Set(r.k(key), value, expiration)
}

// Append 如果 key 已经存在并且是一个字符串， APPEND 命令将 value 追加到 key 原来的值的末尾。
// 如果 key 不存在， APPEND 就简单地将给定 key 设为 value ，就像执行 SET key value 一样。
func (r *Client) Append(key, value string) *redis.IntCmd {
	return r.client.Append(r.k(key), value)
}

// Del 删除给定的一个或多个 key 。
// 不存在的 key 会被忽略。
func (r *Client) Del(keys ...string) *redis.IntCmd {
	return r.client.Del(r.ks(keys...)...)
}

// Unlink 这个命令非常类似于DEL：它删除指定的键。就像DEL键一样，如果它不存在，它将被忽略。但是，该命令在不同的线程中执行实际的内存回收，所以它不会阻塞，而DEL是。这是命令名称的来源：命令只是将键与键空间断开连接。实际删除将在以后异步发生。
func (r *Client) Unlink(keys ...string) *redis.IntCmd {
	return r.client.Unlink(r.ks(keys...)...)
}

// MSet 同时设置一个或多个 key-value 对。
func (r *Client) MSet(values ...interface{}) *redis.StatusCmd {
	return r.client.MSet(values...)
}

// MSetNX 同时设置一个或多个 key-value 对，当且仅当所有给定 key 都不存在。
// 即使只有一个给定 key 已存在， MSETNX 也会拒绝执行所有给定 key 的设置操作。
// MSETNX 是原子性的，因此它可以用作设置多个不同 key 表示不同字段(field)的唯一性逻辑对象(unique logic object)，所有字段要么全被设置，要么全不被设置。
func (r *Client) MSetNX(values ...interface{}) *redis.BoolCmd {
	return r.client.MSetNX(values...)
}

// SAdd 将一个或多个 member 元素加入到集合 key 当中，已经存在于集合的 member 元素将被忽略。
// 假如 key 不存在，则创建一个只包含 member 元素作成员的集合。
// 当 key 不是集合类型时，返回一个错误。
func (r *Client) SAdd(key string, members ...interface{}) *redis.IntCmd {
	return r.client.SAdd(r.k(key), members...)
}

// SCard 返回集合 key 的基数(集合中元素的数量)。
func (r *Client) SCard(key string) *redis.IntCmd {
	return r.client.SCard(r.k(key))
}
func (r *Client) SDiff(keys ...string) *redis.StringSliceCmd {
	return r.client.SDiff(r.ks(keys...)...)
}
func (r *Client) SDiffStore(destination string, keys ...string) *redis.IntCmd {
	return r.client.SDiffStore(r.k(destination), r.ks(keys...)...)
}
func (r *Client) SInter(keys ...string) *redis.StringSliceCmd {
	return r.client.SInter(r.ks(keys...)...)
}
func (r *Client) SInterStore(destination string, keys ...string) *redis.IntCmd {
	return r.client.SInterStore(r.k(destination), r.ks(keys...)...)
}
func (r *Client) SIsMember(key string, member interface{}) *redis.BoolCmd {
	return r.client.SIsMember(r.k(key), member)
}
func (r *Client) SMembers(key string) *redis.StringSliceCmd {
	return r.client.SMembers(r.k(key))
}
func (r *Client) SMove(source, destination string, member interface{}) *redis.BoolCmd {
	return r.client.SMove(r.k(source), r.k(destination), member)
}
func (r *Client) SPop(key string) *redis.StringCmd {
	return r.client.SPop(r.k(key))
}
func (r *Client) SPopN(key string, count int64) *redis.StringSliceCmd {
	return r.client.SPopN(r.k(key), count)
}
func (r *Client) SRandMember(key string) *redis.StringCmd {
	return r.client.SRandMember(r.k(key))
}
func (r *Client) SRandMemberN(key string, count int64) *redis.StringSliceCmd {
	return r.client.SRandMemberN(r.k(key), count)
}
func (r *Client) SRem(key string, members ...interface{}) *redis.IntCmd {
	return r.client.SRem(r.k(key), members...)
}
func (r *Client) SUnion(keys ...string) *redis.StringSliceCmd {
	return r.client.SUnion(r.ks(keys...)...)
}
func (r *Client) SUnionStore(destination string, keys ...string) *redis.IntCmd {
	return r.client.SUnionStore(r.k(destination), r.ks(keys...)...)
}

// -------------- SortedSettable

func (r *Client) ZAdd(key string, members ...*redis.Z) *redis.IntCmd {
	return r.client.ZAdd(r.k(key), members...)
}
func (r *Client) ZAddNX(key string, members ...*redis.Z) *redis.IntCmd {
	return r.client.ZAddNX(r.k(key), members...)
}
func (r *Client) ZAddXX(key string, members ...*redis.Z) *redis.IntCmd {
	return r.client.ZAddXX(r.k(key), members...)
}
func (r *Client) ZAddCh(key string, members ...*redis.Z) *redis.IntCmd {
	return r.client.ZAddCh(r.k(key), members...)
}
func (r *Client) ZAddNXCh(key string, members ...*redis.Z) *redis.IntCmd {
	return r.client.ZAddNXCh(r.k(key), members...)
}
func (r *Client) ZAddXXCh(key string, members ...*redis.Z) *redis.IntCmd {
	return r.client.ZAddXXCh(r.k(key), members...)
}
func (r *Client) ZIncr(key string, member *redis.Z) *redis.FloatCmd {
	return r.client.ZIncr(r.k(key), member)
}
func (r *Client) ZIncrNX(key string, member *redis.Z) *redis.FloatCmd {
	return r.client.ZIncrNX(r.k(key), member)
}
func (r *Client) ZIncrXX(key string, member *redis.Z) *redis.FloatCmd {
	return r.client.ZIncrXX(r.k(key), member)
}
func (r *Client) ZCard(key string) *redis.IntCmd {
	return r.client.ZCard(r.k(key))
}
func (r *Client) ZCount(key, min, max string) *redis.IntCmd {
	return r.client.ZCount(r.k(key), min, max)
}
func (r *Client) ZIncrBy(key string, increment float64, member string) *redis.FloatCmd {
	return r.client.ZIncrBy(r.k(key), increment, member)
}

func (r *Client) ZInterStore(key string, store *redis.ZStore) *redis.IntCmd {
	return r.client.ZInterStore(r.k(key), store)
}
func (r *Client) ZRange(key string, start, stop int64) *redis.StringSliceCmd {
	return r.client.ZRange(r.k(key), start, stop)
}
func (r *Client) ZRangeWithScores(key string, start, stop int64) *redis.ZSliceCmd {
	return r.client.ZRangeWithScores(r.k(key), start, stop)
}
func (r *Client) ZRangeByScore(key string, opt *redis.ZRangeBy) *redis.StringSliceCmd {
	return r.client.ZRangeByScore(r.k(key), opt)
}
func (r *Client) ZRangeByLex(key string, opt *redis.ZRangeBy) *redis.StringSliceCmd {
	return r.client.ZRangeByLex(r.k(key), opt)
}
func (r *Client) ZRangeByScoreWithScores(key string, opt *redis.ZRangeBy) *redis.ZSliceCmd {
	return r.client.ZRangeByScoreWithScores(r.k(key), opt)
}
func (r *Client) ZRank(key, member string) *redis.IntCmd {
	return r.client.ZRank(r.k(key), member)
}
func (r *Client) ZRem(key string, members ...interface{}) *redis.IntCmd {
	return r.client.ZRem(r.k(key), members...)
}
func (r *Client) ZRemRangeByRank(key string, start, stop int64) *redis.IntCmd {
	return r.client.ZRemRangeByRank(r.k(key), start, stop)
}
func (r *Client) ZRemRangeByScore(key, min, max string) *redis.IntCmd {
	return r.client.ZRemRangeByScore(r.k(key), min, max)
}
func (r *Client) ZRemRangeByLex(key, min, max string) *redis.IntCmd {
	return r.client.ZRemRangeByLex(r.k(key), min, max)
}
func (r *Client) ZRevRange(key string, start, stop int64) *redis.StringSliceCmd {
	return r.client.ZRevRange(r.k(key), start, stop)
}
func (r *Client) ZRevRangeWithScores(key string, start, stop int64) *redis.ZSliceCmd {
	return r.client.ZRevRangeWithScores(r.k(key), start, stop)
}
func (r *Client) ZRevRangeByScore(key string, opt *redis.ZRangeBy) *redis.StringSliceCmd {
	return r.client.ZRevRangeByScore(r.k(key), opt)
}
func (r *Client) ZRevRangeByLex(key string, opt *redis.ZRangeBy) *redis.StringSliceCmd {
	return r.client.ZRevRangeByLex(r.k(key), opt)
}
func (r *Client) ZRevRangeByScoreWithScores(key string, opt *redis.ZRangeBy) *redis.ZSliceCmd {
	return r.client.ZRevRangeByScoreWithScores(r.k(key), opt)
}
func (r *Client) ZRevRank(key, member string) *redis.IntCmd {
	return r.client.ZRevRank(r.k(key), member)
}
func (r *Client) ZScore(key, member string) *redis.FloatCmd {
	return r.client.ZScore(r.k(key), member)
}

func (r *Client) ZUnionStore(dest string, store *redis.ZStore) *redis.IntCmd {
	return r.client.ZUnionStore(r.k(dest), store)
}

// -------------- BlockedSettable

func (r *Client) BLPop(timeout time.Duration, keys ...string) *redis.StringSliceCmd {
	return r.client.BLPop(timeout, r.ks(keys...)...)
}
func (r *Client) BRPop(timeout time.Duration, keys ...string) *redis.StringSliceCmd {
	return r.client.BRPop(timeout, r.ks(keys...)...)
}
func (r *Client) BRPopLPush(source, destination string, timeout time.Duration) *redis.StringCmd {
	return r.client.BRPopLPush(r.k(source), r.k(destination), timeout)
}

// -------------- Scanner

func (r *Client) Type(key string) *redis.StatusCmd {
	return r.client.Type(r.k(key))
}
func (r *Client) Scan(cursor uint64, match string, count int64) *redis.ScanCmd {
	return r.client.Scan(cursor, r.k(match), count)
}
func (r *Client) SScan(key string, cursor uint64, match string, count int64) *redis.ScanCmd {
	return r.client.SScan(r.k(key), cursor, match, count)
}
func (r *Client) ZScan(key string, cursor uint64, match string, count int64) *redis.ScanCmd {
	return r.client.ZScan(r.k(key), cursor, match, count)
}
func (r *Client) HScan(key string, cursor uint64, match string, count int64) *redis.ScanCmd {
	return r.client.HScan(r.k(key), cursor, match, count)
}

// -------------- Publisher

func (r *Client) Publish(channel string, message interface{}) *redis.IntCmd {
	return r.client.Publish(r.k(channel), message)
}
func (r *Client) Subscribe(channels ...string) *redis.PubSub {
	return r.client.Subscribe(r.ks(channels...)...)
}

// Pipeline 获取管道
func (r *Client) Pipeline() redis.Pipeliner {
	return r.client.Pipeline()
}

//Pipelined .
func (r *Client) Pipelined(fn func(redis.Pipeliner) error) ([]redis.Cmder, error) {
	return r.Pipelined(fn)
}

// ErrNotImplemented not implemented error
var ErrNotImplemented = errors.New("Not implemented")
