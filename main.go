package main

import (
	"fmt"

	graph "github.com/iamemilio/dependencyFramework/dependencygraph"
)

func main() {
	// Basic test case
	a, _ := graph.NewNode("a")
	b, _ := graph.NewNode("b")
	c, _ := graph.NewNode("c")
	d, _ := graph.NewNode("d")
	e, _ := graph.NewNode("e")
	f, _ := graph.NewNode("f")
	g, _ := graph.NewNode("g")
	h, _ := graph.NewNode("h")

	// a-----b---------c-----g
	//	      \         \
	// d-------e--X--f---h
	b.DependsOn(a)
	c.DependsOn(b)
	e.DependsOn(b)
	e.DependsOn(d)
	f.DependsOn(e)
	g.DependsOn(c)
	h.DependsOnList([]*graph.Node{c, f})

	modules := []*graph.Node{a, b, c, d, e, f}
	roots := graph.GetRootDependencies(modules)
	fmt.Printf("roots: %v\n", graph.ListNodeNames(roots))

	for _, root := range roots {
		err := root.Pass()
		if err != nil {
			fmt.Println(err)
			return
		}
	}

	// Get the first depth of modules
	d1, _ := graph.Step(roots)
	fmt.Printf("depth 1: %v\n", graph.ListNodeNames(d1))

	for _, m := range d1 {
		err := m.Pass()
		if err != nil {
			fmt.Println(err)
			return
		}
	}

	// Get the second depth of modules
	d2, _ := graph.Step(d1)
	fmt.Printf("depth 2: %v\n", graph.ListNodeNames(d2))

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

	d3, _ := graph.Step(d2)
	fmt.Printf("depth 3: %v\n", graph.ListNodeNames(d3))
}
