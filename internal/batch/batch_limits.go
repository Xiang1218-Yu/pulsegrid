package batch

const (
	DefaultBatchSize = 100
	MaxBatchSize     = 1000
	MaxBatchWorkers  = 32
)

func NormalizeWorkers(value int) int {
	if value < 1 {
		return 1
	}
	if value > MaxBatchWorkers {
		return MaxBatchWorkers
	}
	return value
}

func NormalizeSize(value int) int {
	if value < 1 {
		return DefaultBatchSize
	}
	if value > MaxBatchSize {
		return MaxBatchSize
	}
	return value
}
