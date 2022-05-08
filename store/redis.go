package store

type ConfigRedis struct {
	Addrs     []string
	Password  string
	DB        int
	IsCluster bool
}
