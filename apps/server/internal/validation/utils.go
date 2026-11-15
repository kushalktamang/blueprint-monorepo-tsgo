package validation

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/kushalktamang/blueprint-monorepo-tsgo/internal/errs"
)

const maxBodyBytes = 1 << 20 // 1 MiB

type Validatable interface {
	Validate() error
}

type CustomValidationError struct {
	Field   string
	Message string
}

type CustomValidationErrors []CustomValidationError

func (e CustomValidationErrors) Error() string {
	return "custom validation failed"
}

var (
	uuidRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	validate  = newValidator()
)

func newValidator() *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())

	// Report JSON key names instead of Go field names.
	v.RegisterTagNameFunc(func(f reflect.StructField) string {
		name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
		if name == "-" {
			return ""
		}
		return name
	})

	// Custom tag backing the "uuidList" case below.
	_ = v.RegisterValidation("uuidList", func(fl validator.FieldLevel) bool {
		for s := range strings.SplitSeq(fl.Field().String(), ",") {
			if !IsValidUUID(strings.TrimSpace(s)) {
				return false
			}
		}
		return true
	})

	return v
}

// Struct is what your payloads' Validate() methods call.
func Struct(v any) error {
	return validate.Struct(v)
}

func IsValidUUID(s string) bool {
	return uuidRegex.MatchString(s)
}

func BindAndValidate(w http.ResponseWriter, r *http.Request, payload Validatable) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(payload); err != nil {
		return errs.NewBadRequestError(decodeErrMessage(err), false, nil, nil, nil)
	}
	// Reject `{"a":1}{"b":2}` style trailing data.
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errs.NewBadRequestError("body must contain a single JSON object", false, nil, nil, nil)
	}

	if msg, fieldErrors := validateStruct(payload); fieldErrors != nil {
		return errs.NewBadRequestError(msg, true, nil, fieldErrors, nil)
	}
	return nil
}

func decodeErrMessage(err error) string {
	var (
		syntaxErr *json.SyntaxError
		typeErr   *json.UnmarshalTypeError
		maxErr    *http.MaxBytesError
	)
	switch {
	case errors.As(err, &maxErr):
		return "request body too large"
	case errors.As(err, &syntaxErr):
		return fmt.Sprintf("malformed JSON at position %d", syntaxErr.Offset)
	case errors.As(err, &typeErr):
		return fmt.Sprintf("field %q must be of type %s", typeErr.Field, typeErr.Type)
	case errors.Is(err, io.EOF):
		return "request body is empty"
	case errors.Is(err, io.ErrUnexpectedEOF):
		return "malformed JSON: unexpected end of body"
	case strings.HasPrefix(err.Error(), "json: unknown field"):
		return err.Error()
	default:
		return "invalid request body"
	}
}

func validateStruct(v Validatable) (string, []errs.FieldError) {
	if err := v.Validate(); err != nil {
		return extractValidationErrors(err)
	}
	return "", nil
}

func extractValidationErrors(err error) (string, []errs.FieldError) {
	if custom, ok := errors.AsType[CustomValidationErrors](err); ok {
		fe := make([]errs.FieldError, 0, len(custom))
		for _, e := range custom {
			fe = append(fe, errs.FieldError{Field: e.Field, Error: e.Message})
		}
		return "Validation failed", fe
	}

	validationErrors, ok := errors.AsType[validator.ValidationErrors](err)
	if !ok {
		return "Validation failed", []errs.FieldError{{Field: "", Error: "invalid input"}}
	}

	fieldErrors := make([]errs.FieldError, 0, len(validationErrors))

	for _, fe := range validationErrors {
		field := fieldPath(fe)
		var msg string

		switch fe.Tag() {
		case "required":
			msg = "is required"
		case "min":
			if fe.Kind() == reflect.String {
				msg = fmt.Sprintf("must be at least %s characters", fe.Param())
			} else {
				msg = fmt.Sprintf("must be at least %s", fe.Param())
			}
		case "max":
			if fe.Kind() == reflect.String {
				msg = fmt.Sprintf("must not exceed %s characters", fe.Param())
			} else {
				msg = fmt.Sprintf("must not exceed %s", fe.Param())
			}
		case "oneof":
			msg = fmt.Sprintf("must be one of: %s", fe.Param())
		case "email":
			msg = "must be a valid email address"
		case "e164":
			msg = "must be a valid phone number with country code"
		case "uuid":
			msg = "must be a valid UUID"
		case "uuidList":
			msg = "must be a comma-separated list of valid UUIDs"
		default:
			if fe.Param() != "" {
				msg = fmt.Sprintf("failed %s:%s", fe.Tag(), fe.Param())
			} else {
				msg = fmt.Sprintf("failed %s", fe.Tag())
			}
		}

		fieldErrors = append(fieldErrors, errs.FieldError{Field: field, Error: msg})
	}

	return "Validation failed", fieldErrors
}

// fieldPath turns "CreateUser.items[2].name" into "items[2].name".
func fieldPath(fe validator.FieldError) string {
	_, path, found := strings.Cut(fe.Namespace(), ".")
	if !found {
		return fe.Field()
	}
	return path
}
