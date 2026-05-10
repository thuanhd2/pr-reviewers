package executor

import "fmt"

type Registry struct {
	executors map[string]Executor
}

func NewRegistry() *Registry {
	return &Registry{executors: make(map[string]Executor)}
}

func (r *Registry) Register(exec Executor) {
	r.executors[exec.Name()] = exec
}

func (r *Registry) Get(name string) (Executor, error) {
	exec, ok := r.executors[name]
	if !ok {
		return nil, fmt.Errorf("executor %q not found", name)
	}
	return exec, nil
}

func (r *Registry) List() []string {
	names := make([]string, 0, len(r.executors))
	for name := range r.executors {
		names = append(names, name)
	}
	return names
}
