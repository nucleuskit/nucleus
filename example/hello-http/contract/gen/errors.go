package gen

// Code is a generated Nucleus error code.
type Code int

const (
	CodeInvalidName      Code = 4001
	CodeGreetingNotFound Code = 4041
)

// ErrorMessages maps generated error codes to stable messages.
var ErrorMessages = map[Code]string{
	CodeInvalidName:      "invalid_name",
	CodeGreetingNotFound: "greeting_not_found",
}

// HTTPStatuses maps generated error codes to HTTP statuses.
var HTTPStatuses = map[Code]int{
	CodeInvalidName:      400,
	CodeGreetingNotFound: 404,
}
