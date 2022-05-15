package util

import (
	"strings"
)

type TrieNode struct {
	Routing   any
	Childrens []*TrieNode
	Value     any
}

func (n *TrieNode) AddChildren(routing any, value any) {
	nc := n.ChildrenInfo(routing)
	if nc != nil {
		nc.Value = value
		return
	}
	n.Childrens = append(n.Childrens, &TrieNode{
		Routing: routing,
		Value:   value,
	})
}

func (n *TrieNode) ChildrenInfo(routing any) *TrieNode {
	for _, v := range n.Childrens {
		if v.Routing == routing {
			return v
		}
	}
	return nil
}

func (n *TrieNode) Iteration() {

}

type TrieRoots struct {
	N         []*TrieNode
	EqualF    func(r1, r2 any) bool
	RoutingsF func(r string) []any
}

func (r *TrieRoots) Add(routings []any, value any) {
	ls := r.N
	var fNode *TrieNode
	for i, routing := range routings {
		fn, _ := r.info(routing, ls)
		if fn == nil {
			fn = &TrieNode{
				Routing: routing,
			}
			if fNode == nil {
				r.N = append(r.N, fn)
			} else {
				fNode.Childrens = append(fNode.Childrens, fn)
			}
		}
		if i == len(routings)-1 {
			fn.Value = value
		}
		fNode = fn
		ls = fNode.Childrens
	}
}

func (r *TrieRoots) Info(routings []any) *TrieNode {
	ls := r.N
	var fNode *TrieNode
	for i, routing := range routings {
		fn, _ := r.info(routing, ls)
		if fn == nil {
			return nil
		}
		if i == len(routings)-1 {
			return fn
		}
		fNode = fn
		ls = fNode.Childrens
	}
	return nil
}

func (r *TrieRoots) InfoExist(routings []any) *TrieNode {
	ls := r.N
	var fNode *TrieNode
	for i, routing := range routings {
		fn, _ := r.info(routing, ls)
		if fn == nil {
			if fNode != nil {
				return fNode
			}
			return nil
		}
		if i == len(routings)-1 {
			return fn
		}
		fNode = fn
		ls = fNode.Childrens
	}
	return nil
}

func (r *TrieRoots) AddFromRouting(rg string, value any) {
	if r.RoutingsF == nil {
		panic("RoutingsF is empty")
	}
	rs := r.RoutingsF(rg)
	r.Add(rs, value)
}

func (r *TrieRoots) InfoFromRouting(rg string) *TrieNode {
	if r.RoutingsF == nil {
		panic("RoutingsF is empty")
	}
	return r.Info(r.RoutingsF(rg))
}

func (r *TrieRoots) InfoExistFromRouting(rg string) *TrieNode {
	if r.RoutingsF == nil {
		panic("RoutingsF is empty")
	}
	return r.InfoExist(r.RoutingsF(rg))
}

func (r *TrieRoots) Remove(routings []any) *TrieNode {
	ls := r.N
	var fNode *TrieNode
	for i, routing := range routings {
		fn, idx := r.info(routing, ls)
		if fn == nil {
			return nil
		}
		if i == len(routings)-1 {
			fNode.Childrens = append(fNode.Childrens[:idx], fNode.Childrens[idx+1:len(fNode.Childrens)]...)
			return fn
		}
	}
	return nil
}

func (r *TrieRoots) RemoveFromRouting(rg string) *TrieNode {
	return r.Remove(r.RoutingsF(rg))
}

func (r *TrieRoots) info(routing any, ls []*TrieNode) (*TrieNode, int) {
	for i, n := range ls {
		if r.EqualF != nil && r.EqualF(n.Routing, routing) {
			return n, i
		} else if n.Routing == routing {
			return n, i
		}
	}
	return nil, -1
}

func NewTrieRootsFromMap(m map[string]string, seq string) *TrieRoots {
	r := &TrieRoots{
		EqualF: func(r1, r2 any) bool {
			if r1.(string) == r2.(string) {
				return !false
			}
			return false
		},
		RoutingsF: func(rv string) []any {
			ls := strings.Split(rv, seq)
			las := []any{}
			for _, v := range ls {
				las = append(las, v)
			}
			return las
		},
	}
	for key, v := range m {
		r.AddFromRouting(key, v)
	}
	return r
}
