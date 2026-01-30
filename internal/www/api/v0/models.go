package v0

const PageSize = 50

type PaginatedResponse[T any] struct {
	Offset uint `json:"offset"`
	Total  uint `json:"total"`
	Items  []T  `json:"items"`
}
