package dependencygraph

import (
	"errors"
	"fmt"
	"sync"
)

type status struct {
	Failed bool
}

type Node struct {
	name            string // name must be a unique field
	blocked         bool   // A node is blocked when a dependency fails, making it unable to run
	hasDependencies bool   // a flag that tracks if a node has any dependencies
	depsRemaining   int    // a count of the dependencies that the current node is waiting on
	sync.Mutex
	dependencyOf []*Node // a list of pointers to nodes the current node is a dependency of
	*status              // nil = not run, false = passed, true = failed
}

func NewNode(name string) *Node {
	return &Node{name: name}
}

func (m *Node) Pass() error {
	m.Lock()
	defer m.Unlock()
	if m.status != nil {
		return fmt.Errorf("you cannot apply results to a node multiple times")
	}
	m.status = &status{Failed: false}

	for _, dep := range m.dependencyOf {
		dep.Lock()
		dep.depsRemaining -= 1
		dep.Unlock()
	}
	return nil
}

func (m *Node) Fail() error {
	m.Lock()
	defer m.Unlock()
	if m.status != nil {
		return fmt.Errorf("you cannot apply results to a node multiple times")
	}
	m.status = &status{Failed: true}

	for _, dep := range m.dependencyOf {
		dep.Lock()
		dep.depsRemaining -= 1
		dep.Unlock()
	}

	// Mark all dependencies of node as blocked
	// Walks the whole graph and marks all dependencies affected as blocked
	nodes := New()
	for _, mod := range m.dependencyOf {
		nodes.Push(mod)
	}

	visited := make(map[string]bool)
	visited[m.name] = true // prevents mutex blocking in the case of a circular dependency

	for nodes.Len() != 0 {
		node, _ := nodes.Pop()
		if node.name == m.name {
			return fmt.Errorf("graph has circular dependency")
		}
		if !visited[node.name] {
			visited[node.name] = true
			node.blocked = true // effectively removes the node from the graph
			for _, dependency := range node.dependencyOf {
				if !visited[dependency.name] {
					nodes.Push(dependency)
				}
			}
		}
	}
	return nil
}

func (m *Node) DependsOn(dep *Node) error {
	if dep.name == m.name {
		return errors.New("a node cannot be dependent on itself")
	}

	dep.Lock()
	for _, d := range dep.dependencyOf {
		if d.name == dep.name {
			return fmt.Errorf("node dependency %s is a duplicate", dep.name)
		}
	}
	dep.dependencyOf = append(dep.dependencyOf, m)
	dep.Unlock()

	m.Lock()
	m.depsRemaining += 1
	m.hasDependencies = true
	m.Unlock()
	return nil
}

// DependsOnList is a convenience function that allows you to pass a list of nodes
// to add to a node's dependencies
func (m *Node) DependsOnList(deps []*Node) error {
	seen := map[string]bool{}
	for _, dep := range deps {
		if seen[dep.name] {
			return fmt.Errorf("%s is a duplicate dependency", dep.name)
		}
		seen[dep.name] = true
		err := m.DependsOn(dep)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetDependencyNames returns the names of the nodes the current node
// is a dependency of as a list of strings
// For debugging purposes
// Not Thread Safe
func (m *Node) GetDependencyNames() (names []string) {
	for _, dep := range m.dependencyOf {
		dep.Lock()
		names = append(names, dep.name)
		dep.Unlock()
	}
	return names
}

// GetRootDependencies returns a list of nodes that do not have any dependencies
// This does not need to be thread safe, since the `hasDependencies` property can only be set to true
// This method is intended to be run in sequential code blocks only, after nodes have already been
// initialized and had their dependencies set
func GetRootDependencies(nodes []*Node) (roots []*Node) {
	for _, node := range nodes {
		if !node.hasDependencies {
			roots = append(roots, node)
		}
	}
	return roots
}

// Step returns all the nodes at the next depth from the nodes in the list `nodes`
// In order for this to work sproperly, all the nodes passed to this function
// must have been run
// This function is not thread safe because it should never be run in a concurrent thread
func Step(nodes []*Node) (depth []*Node, err error) {
	for _, node := range nodes {
		if node.status == nil {
			return nil, fmt.Errorf("node %s status was not updated, can not step", node.name)
		}

		for _, dependency := range node.dependencyOf {
			if !dependency.blocked && dependency.depsRemaining == 0 {
				depth = append(depth, dependency)
			}
		}
	}
	return depth, nil
}

// ListNodeNames prints the names of all nodes in a list
// this is a convenience function that is primarily for debugging use
func ListNodeNames(nodes []*Node) (a []string) {
	for _, node := range nodes {
		a = append(a, node.name)
	}
	return a
}
