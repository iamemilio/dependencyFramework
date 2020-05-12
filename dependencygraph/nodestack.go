package dependencygraph

import (
	"errors"
	"sync"
)

// nodeStack is a simple stack object that can hold dependencyGraph nodes
// this is not intended to be a user facing library and is primarily a
// data structure that facilitates the dependencyGraph
type nodeStack struct {
	sync.Mutex
	s []*Node
}

func New() *nodeStack {
	return &nodeStack{sync.Mutex{}, make([]*Node, 0)}
}

func (s *nodeStack) Push(m *Node) {
	s.Lock()
	defer s.Unlock()

	s.s = append(s.s, m)
}

func (s *nodeStack) Pop() (*Node, error) {
	s.Lock()
	defer s.Unlock()

	l := len(s.s)
	if l == 0 {
		return nil, errors.New("Empty Stack")
	}

	res := s.s[l-1]
	s.s = s.s[:l-1]
	return res, nil
}

func (s *nodeStack) Len() int {
	return len(s.s)
}
