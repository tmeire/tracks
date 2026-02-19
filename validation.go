package tracks

import (
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

var emailRegex = regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,4}$`)

// ValidationErrors is a map of field names to error messages.
type ValidationErrors map[string][]string

func (v ValidationErrors) Error() string {
	return "validation failed"
}

func (v ValidationErrors) FieldErrors() map[string][]string {
	return v
}

// ParseAndValidate parses the request and validates the populated struct.
func ParseAndValidate(r *http.Request, v any) error {
	if err := ParseRequest(r, v); err != nil {
		return err
	}
	return Validate(v)
}

// Validate validates a struct based on 'validate' tags.
func Validate(v any) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}

	if rv.Kind() != reflect.Struct {
		return nil
	}

	errors := make(ValidationErrors)
	rt := rv.Type()

	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		tag := field.Tag.Get("validate")
		if tag == "" {
			continue
		}

		fieldName := field.Tag.Get("json")
		if fieldName == "" {
			fieldName = field.Tag.Get("form")
		}
		if fieldName == "" || fieldName == "-" {
			fieldName = field.Name
		}

		val := rv.Field(i)
		rules := strings.Split(tag, ",")

		for _, rule := range rules {
			if err := validateField(fieldName, val, rule, rv); err != nil {
				errors[fieldName] = append(errors[fieldName], err.Error())
			}
		}
	}

	if len(errors) > 0 {
		return errors
	}
	return nil
}

func validateField(name string, val reflect.Value, rule string, obj reflect.Value) error {
	parts := strings.Split(rule, "=")
	ruleName := parts[0]
	var ruleVal string
	if len(parts) > 1 {
		ruleVal = parts[1]
	}

	switch ruleName {
	case "required":
		if isZero(val) {
			return fmt.Errorf("field is required")
		}
	case "email":
		if val.Kind() == reflect.String && val.String() != "" {
			if !emailRegex.MatchString(strings.ToLower(val.String())) {
				return fmt.Errorf("invalid email format")
			}
		}
	case "min":
		min, _ := strconv.Atoi(ruleVal)
		switch val.Kind() {
		case reflect.String:
			if len(val.String()) < min {
				return fmt.Errorf("minimum length is %d", min)
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if val.Int() < int64(min) {
				return fmt.Errorf("minimum value is %d", min)
			}
		}
	case "max":
		max, _ := strconv.Atoi(ruleVal)
		switch val.Kind() {
		case reflect.String:
			if len(val.String()) > max {
				return fmt.Errorf("maximum length is %d", max)
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if val.Int() > int64(max) {
				return fmt.Errorf("maximum value is %d", max)
			}
		}
	case "eqfield":
		otherField := obj.FieldByName(ruleVal)
		if otherField.IsValid() {
			if val.Interface() != otherField.Interface() {
				return fmt.Errorf("does not match %s", ruleVal)
			}
		}
	}

	return nil
}

func isZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String:
		return v.Len() == 0
	case reflect.Ptr, reflect.Interface, reflect.Slice, reflect.Map:
		return v.IsNil()
	default:
		return v.IsZero()
	}
}
