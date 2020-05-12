package main

import (
	"errors"
	"fmt"
	"sync"
)

// ModuleStack

type moduleStack struct {
	lock sync.Mutex
	s    []*Module
}

func NewModuleStack() *moduleStack {
	return &moduleStack{sync.Mutex{}, make([]*Module, 0)}
}

func (s *moduleStack) Push(m *Module) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.s = append(s.s, m)
}

func (s *moduleStack) Pop() (*Module, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	l := len(s.s)
	if l == 0 {
		return nil, errors.New("Empty Stack")
	}

	res := s.s[l-1]
	s.s = s.s[:l-1]
	return res, nil
}

func (s *moduleStack) Len() int {
	return len(s.s)
}

// Modules
type status struct {
	Failed bool
}

type Module struct {
	name            string     // name must be a unique field
	blocked         bool       // A module is blocked when a dependency fails, making it unable to run
	hasDependencies bool       // a flag that tracks if a module has any dependencies
	depsRemaining   int        // a count of the dependencies that the current module is waiting on
	lock            sync.Mutex // TODO: test thread safety
	dependencyOf    []*Module  // a list of pointers to modules the current module is a dependency of
	*status                    // nil = not run, false = passed, true = failed
}

func NewModule(name string) (*Module, error) {
	if name == "" {
		return nil, errors.New("Module must have a name")
	}
	return &Module{name: name}, nil
}

func (m *Module) Pass() error {
	m.lock.Lock()
	defer m.lock.Unlock()
	if m.status != nil {
		return fmt.Errorf("you cannot apply results to a module multiple times")
	}
	m.status = &status{Failed: false}

	for _, dep := range m.dependencyOf {
		dep.lock.Lock()
		dep.depsRemaining -= 1
		dep.lock.Unlock()
	}
	return nil
}

func (m *Module) Fail() error {
	m.lock.Lock()
	defer m.lock.Unlock()
	if m.status != nil {
		return fmt.Errorf("you cannot apply results to a module multiple times")
	}
	m.status = &status{Failed: true}

	for _, dep := range m.dependencyOf {
		dep.lock.Lock()
		dep.depsRemaining -= 1
		dep.lock.Unlock()
	}

	// Mark all dependencies of module as blocked
	// Walks the whole graph and marks all dependencies affected as blocked
	modules := NewModuleStack()
	for _, mod := range m.dependencyOf {
		modules.Push(mod)
	}

	visited := make(map[string]bool)
	visited[m.name] = true

	for modules.Len() != 0 {
		module, _ := modules.Pop()
		if module.name == m.name {
			return fmt.Errorf("graph has circular dependency")
		}
		if !visited[module.name] {
			visited[module.name] = true
			module.blocked = true // effectively removes the node from the graph
			for _, dependency := range module.dependencyOf {
				if !visited[dependency.name] {
					modules.Push(dependency)
				}
			}
		}
	}
	return nil
}

func (m *Module) DependsOn(dep *Module) error {
	if dep.name == m.name {
		return errors.New("a module cannot be dependent on itself")
	}

	dep.lock.Lock()
	for _, d := range dep.dependencyOf {
		if d.name == dep.name {
			return fmt.Errorf("module dependency %s is a duplicate", dep.name)
		}
	}
	dep.dependencyOf = append(dep.dependencyOf, m)
	dep.lock.Unlock()

	m.lock.Lock()
	m.depsRemaining += 1
	m.hasDependencies = true
	m.lock.Unlock()
	return nil
}

func (m *Module) DependsOnList(deps []*Module) error {
	for _, dep := range deps {
		err := m.DependsOn(dep)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetDependencyNames returns the names of the modules the current module
// is a dependency of as a list of strings
// For debugging purposes
// Not Thread Safe
func (m *Module) GetDependencyNames() (names []string) {
	for _, dep := range m.dependencyOf {
		dep.lock.Lock()
		names = append(names, dep.name)
		dep.lock.Unlock()
	}
	return names
}

// GetRootDependencies returns a list of modules that do not have any dependencies
// This does not need to be thread safe, since the `hasDependencies` property can only be set to true
// This method is intended to be run in sequential code blocks only, after modules have already been
// initialized and had their dependencies set
func GetRootDependencies(modules []*Module) (roots []*Module) {
	for _, module := range modules {
		if !module.hasDependencies {
			roots = append(roots, module)
		}
	}
	return roots
}

// Step returns all the modules at the next depth from the modules in the list `modules`
// In order for this to work sproperly, all the modules passed to this function
// must have been run
// This function is not thread safe because it should never be run in a concurrent thread
func Step(modules []*Module) (depth []*Module, err error) {
	for _, module := range modules {
		if module.status == nil {
			return nil, fmt.Errorf("module %s status was not updated, can not step", module.name)
		}

		for _, dependency := range module.dependencyOf {
			if !dependency.blocked && dependency.depsRemaining == 0 {
				depth = append(depth, dependency)
			}
		}
	}
	return depth, nil
}

// ListModuleNames prints the names of all modules in a list
// this is a convenience function that is primarily for debugging use
func listModuleNames(modules []*Module) (a []string) {
	for _, module := range modules {
		a = append(a, module.name)
	}
	return a
}

func main() {
	// Basic test case
	a, _ := NewModule("a")
	b, _ := NewModule("b")
	c, _ := NewModule("c")
	d, _ := NewModule("d")
	e, _ := NewModule("e")
	f, _ := NewModule("f")
	g, _ := NewModule("g")
	h, _ := NewModule("h")

	// a-----b---------c-----g
	//	      \         \
	// d-------e--X--f---h
	b.DependsOn(a)
	c.DependsOn(b)
	e.DependsOn(b)
	e.DependsOn(d)
	f.DependsOn(e)
	g.DependsOn(c)
	h.DependsOnList([]*Module{c, f})

	modules := []*Module{a, b, c, d, e, f}
	roots := GetRootDependencies(modules)
	fmt.Printf("roots: %v\n", listModuleNames(roots))

	for _, root := range roots {
		err := root.Pass()
		if err != nil {
			fmt.Println(err)
			return
		}
	}

	// Get the first depth of modules
	d1, _ := Step(roots)
	fmt.Printf("depth 1: %v\n", listModuleNames(d1))

	for _, m := range d1 {
		err := m.Pass()
		if err != nil {
			fmt.Println(err)
			return
		}
	}

	// Get the second depth of modules
	d2, _ := Step(d1)
	fmt.Printf("depth 2: %v\n", listModuleNames(d2))

	err := c.Pass()
	if err != nil {
		fmt.Println(err)
		return
	}

	err = e.Fail()
	if err != nil {
		fmt.Println(err)
		return
	}

	d3, _ := Step(d2)
	fmt.Printf("depth 3: %v\n", listModuleNames(d3))
}
