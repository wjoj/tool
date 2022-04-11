package filter

type Filter interface {
	Insert(msg []byte) bool
	Lookup(msg []byte) bool
}
