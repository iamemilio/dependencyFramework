package main

import (
	"fmt"

	graph "github.com/iamemilio/dependencyFramework/dependencygraph"
)

type job func() ([]error, error)

type jobNode struct {
	job
	node *graph.Node
}

// NewJob creates a new job
// each job represents a node in a DAG based on their dependencies
// the job argument is a placeholder for any function that returns only error
// the name must be a unique, non empty string. We cant enforce the uniqueness
// but if there are duplicates, it will cause the library to fail
// TODO(egarcia): enforce unique names
func NewJob(j job, name string) (*jobNode, error) {
	if name == "" {
		return nil, fmt.Errorf("your job must have a unique, non empty name")
	}

	return &jobNode{job: j, node: graph.NewNode(name)}, nil
}

// DependsOn allows you to specify a list of dependencies for a job
// all dependencies must be unique, and valid
func (j *jobNode) DependsOn(jobs []*jobNode) error {
	nodeList := []*graph.Node{}
	for _, job := range jobs {
		nodeList = append(nodeList, job.node)
	}

	return j.node.DependsOnList(nodeList)
}

// Run runs the the specified job. If this job fails, it will prevent all
// jobs that are dependent on it from running
func (j *jobNode) Run() ([]error, error) {
	res, err := j.job()

	// Case: job had an error
	if err != nil {
		e := j.node.Fail()
		if e != nil {
			return nil, err
		}
		return nil, err
	}

	// Case: job ran succesfully but at least one validation failed
	if res != nil && len(res) > 0 {
		e := j.node.Fail()
		if e != nil {
			return nil, e
		}
		return res, nil
	}

	// Case: job ran succesfully and no validations failed
	e := j.node.Pass()
	if e != nil {
		return nil, e
	}
	return nil, nil
}
