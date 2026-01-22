package v0

const PageSize = 50

type Paginated[T any] struct {
	Offset uint `json:"offset"`
	Total  uint `json:"total"`
	Items  []T  `json:"items"`
}
