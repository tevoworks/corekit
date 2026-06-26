package validation

import (
	"fmt"
	"net/http"
	"net/mail"
	"net/url"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

// NewEchoValidator returns an echo.Validator backed by go-playground/validator/v10
// with custom rules: nohtml, allowhtml, urlstrict.
func NewEchoValidator() echo.Validator {
	v := validator.New()

	// nohtml: rejects strings containing < or > characters
	_ = v.RegisterValidation("nohtml", func(fl validator.FieldLevel) bool {
		s, ok := fl.Field().Interface().(string)
		if !ok {
			return true
		}
		return !strings.ContainsAny(s, "<>")
	})

	// allowhtml: pass-through marker (no validation, but documents intent)
	_ = v.RegisterValidation("allowhtml", func(fl validator.FieldLevel) bool {
		return true
	})

	// urlstrict: validates URL with scheme + host
	_ = v.RegisterValidation("urlstrict", func(fl validator.FieldLevel) bool {
		s, ok := fl.Field().Interface().(string)
		if !ok || s == "" {
			return true
		}
		u, err := url.Parse(s)
		if err != nil || u.Scheme == "" || u.Host == "" {
			return false
		}
		return u.Scheme == "http" || u.Scheme == "https"
	})

	// emailfmt: validates email format via net/mail
	_ = v.RegisterValidation("emailfmt", func(fl validator.FieldLevel) bool {
		s, ok := fl.Field().Interface().(string)
		if !ok || s == "" {
			return true
		}
		_, err := mail.ParseAddress(s)
		return err == nil
	})

	// password: requires min 8 chars, at least 1 uppercase, 1 lowercase, 1 digit, 1 special char
	_ = v.RegisterValidation("password", func(fl validator.FieldLevel) bool {
		s, ok := fl.Field().Interface().(string)
		if !ok {
			return true
		}
		if len(s) < 8 {
			return false
		}
		var hasUpper, hasLower, hasDigit, hasSpecial bool
		for _, c := range s {
			switch {
			case c >= 'A' && c <= 'Z':
				hasUpper = true
			case c >= 'a' && c <= 'z':
				hasLower = true
			case c >= '0' && c <= '9':
				hasDigit = true
			default:
				hasSpecial = true
			}
		}
		return hasUpper && hasLower && hasDigit && hasSpecial
	})

	return &customValidator{validator: v}
}

type customValidator struct {
	validator *validator.Validate
}

func (cv *customValidator) Validate(i interface{}) error {
	if err := cv.validator.Struct(i); err != nil {
		var msgs []string
		if errs, ok := err.(validator.ValidationErrors); ok {
			for _, fe := range errs {
				msgs = append(msgs, friendlyMessage(fe))
			}
		}
		return fmt.Errorf("%s", strings.Join(msgs, "; "))
	}
	return nil
}

func friendlyMessage(fe validator.FieldError) string {
	field := fe.Field()
	tag := fe.Tag()
	switch tag {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "emailfmt":
		return fmt.Sprintf("%s must be a valid email", field)
	case "password":
		return fmt.Sprintf("%s must be at least 8 characters with uppercase, lowercase, digit, and special character", field)
	case "nohtml":
		return fmt.Sprintf("%s must not contain HTML tags", field)
	case "min":
		return fmt.Sprintf("%s must be at least %s characters", field, fe.Param())
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s", field, fe.Param())
	default:
		return fmt.Sprintf("%s: %s", field, tag)
	}
}

// BindAndValidate calls c.Bind + c.Validate in one step.
// Returns an error for Echo's HTTPErrorHandler to format.
func BindAndValidate(c echo.Context, req interface{}) error {
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}
	if err := c.Validate(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return nil
}
