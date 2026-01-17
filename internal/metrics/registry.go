package metrics

import "github.com/RajarshiParmar/groundtruth/internal/model"

type Metric interface {
	Name() string
	Compute(commits []model.ClassifiedCommit) Result
}

type Registry struct {
	metrics map[string]Metric
}

func NewRegistry() *Registry {
	return &Registry{
		metrics: make(map[string]Metric),
	}
}

func (r *Registry) Register(m Metric) {
	r.metrics[m.Name()] = m
}

func (r *Registry) Get(name string) (Metric, bool) {
	m, ok := r.metrics[name]
	return m, ok
}
