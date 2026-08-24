package jobs

type WorkerConfig struct {
	Name     string `json:"name"`
	Workers  int    `json:"workers"`
	Capacity int    `json:"capacity"`
	MaxRetry int    `json:"max_retry"`
	Enabled  bool   `json:"enabled"`
}

func DefaultWorkerConfig(name string) WorkerConfig {
	return WorkerConfig{Name: name, Workers: 1, Capacity: 32, MaxRetry: 1, Enabled: true}
}
