package httpapi

type CollectionPayload[T any] struct {
	Items []T `json:"items"`
	Count int `json:"count"`
}

func NewCollection[T any](items []T) CollectionPayload[T] {
	return CollectionPayload[T]{Items: items, Count: len(items)}
}

type PagePayload[T any] struct {
	Items []T `json:"items"`
	Page  int `json:"page"`
	Size  int `json:"size"`
	Total int `json:"total"`
}
