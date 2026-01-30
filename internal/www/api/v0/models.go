package v0

const PageSize = 50

type PaginatedResponse[T any] struct {
	Offset uint `json:"offset"`
	Total  uint `json:"total"`
	Items  []T  `json:"items"`
}

//

type ErrorResponse struct {
	Errors []Error `json:"errors"`
}

type Error struct {
	Error   string `json:"error"`         // Error code to uniquely identify an error.
	Message string `json:"message"`       // Error message to display to user in English.
	Key     string `json:"key,omitempty"` // Optional key identifying the field causing the issue.
}
