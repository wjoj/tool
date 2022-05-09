package limit

type Limit interface {
	Allow() bool
}
