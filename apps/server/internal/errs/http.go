package errs

import "strings"

type FieldError struct {
	Field string `json:"field"`
	Error string `json:"error"`
}

type ActionType string

type Action struct {
	Type    ActionType `json:"type"`
	Message string     `json:"message"`
	Value   string     `json:"value"`
}

type HTTPError struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	Status   int    `json:"status"`
	Override bool   `json:"override"`
	// Field level errors
	Errors []FieldError `json:"errors"`
	// Action errors
	Action *Action `json:"action"`
}

func (error *HTTPError) Error() string {
	return error.Message
}

// Is reports whether target is an *HTTPError with the same Code, so
// errors.Is can distinguish between different HTTPError kinds instead
// of matching any two HTTPErrors regardless of what they represent.
func (error *HTTPError) Is(target error) bool {
	trg, ok := target.(*HTTPError)
	if !ok {
		return false
	}

	return error.Code == trg.Code
}

func (error *HTTPError) WithMessage(message string) *HTTPError {
	return &HTTPError{
		Code:     error.Code,
		Message:  message,
		Status:   error.Status,
		Override: error.Override,
		Errors:   error.Errors,
		Action:   error.Action,
	}
}

func MakeUpperCaseWithUnderscores(str string) string {
	return strings.ToUpper(strings.ReplaceAll(str, " ", "_"))
}
