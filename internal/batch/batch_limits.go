package batch

const (
	DefaultBatchSize = 100
	MaxBatchSize     = 1000
	MaxBatchWorkers  = 32
)

func NormalizeSize(value int) int {
	if value < 1 {
		return DefaultBatchSize
	}
	if value > MaxBatchSize {
		return MaxBatchSize
	}
	return value
}
