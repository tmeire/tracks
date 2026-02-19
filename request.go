package tracks

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
)

// ParseRequest parses the request body based on the Content-Type header.
// It supports application/json and application/x-www-form-urlencoded.
func ParseRequest(r *http.Request, v any) error {
	contentType := r.Header.Get("Content-Type")

	if strings.Contains(contentType, "application/json") {
		return json.NewDecoder(r.Body).Decode(v)
	}

	// Default to form parsing
	if err := r.ParseForm(); err != nil {
		return err
	}

	return UnmarshalForm(r.PostForm, v)
}

// UnmarshalForm populates a struct from form values using 'form' tags.
func UnmarshalForm(values map[string][]string, v any) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("v must be a pointer to a struct")
	}

	rv = rv.Elem()
	rt := rv.Type()

	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		tag := field.Tag.Get("form")
		if tag == "" || tag == "-" {
			continue
		}

		val, ok := values[tag]
		if !ok || len(val) == 0 {
			continue
		}

		fieldValue := rv.Field(i)
		if !fieldValue.CanSet() {
			continue
		}

		switch field.Type.Kind() {
		case reflect.String:
			fieldValue.SetString(val[0])
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			intVal, err := strconv.ParseInt(val[0], 10, 64)
			if err != nil {
				return fmt.Errorf("failed to parse %s as int: %w", tag, err)
			}
			fieldValue.SetInt(intVal)
		case reflect.Bool:
			boolVal, err := strconv.ParseBool(val[0])
			if err != nil {
				return fmt.Errorf("failed to parse %s as bool: %w", tag, err)
			}
			fieldValue.SetBool(boolVal)
		case reflect.Float32, reflect.Float64:
			floatVal, err := strconv.ParseFloat(val[0], 64)
			if err != nil {
				return fmt.Errorf("failed to parse %s as float: %w", tag, err)
			}
			fieldValue.SetFloat(floatVal)
		case reflect.Slice:
			if field.Type.Elem().Kind() == reflect.String {
				fieldValue.Set(reflect.ValueOf(val))
			}
		}
	}

	return nil
}

// ParseJSONOrForm is a helper that returns a populated struct of type T from the request.
func ParseJSONOrForm[T any](r *http.Request) (T, error) {
	var v T
	
	// If T is a pointer, we need to allocate it
	rv := reflect.ValueOf(&v).Elem()
	if rv.Kind() == reflect.Ptr {
		rv.Set(reflect.New(rv.Type().Elem()))
		err := ParseRequest(r, rv.Interface())
		return v, err
	}
	
	err := ParseRequest(r, &v)
	return v, err
}
