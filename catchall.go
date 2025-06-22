package tracks

import (
	"fmt"
	"net/http"
	"runtime/debug"
)

func CatchAll(handler http.Handler) (http.Handler, error) {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		defer func() {
			v := recover()

			if v == nil {
				return
			}

			switch t := v.(type) {
			case error:
				fmt.Printf("recovered an error panic: %v", t)
			}

			debug.PrintStack()

			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Something went wrong: %s", v)
		}()

		handler.ServeHTTP(w, req)
	}), nil
}
