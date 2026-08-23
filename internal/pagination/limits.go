package pagination

const (
	DefaultPageSize = 25
	MaxPageSize     = 500
)

func NormalizePageSize(value int) int {
	if value < 1 {
		return DefaultPageSize
	}
	if value > MaxPageSize {
		return MaxPageSize
	}
	return value
}
