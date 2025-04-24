package tracks

import "net/http"

type Middleware func(h http.Handler) http.Handler

type Middlewares []Middleware

func (ms Middlewares) Wrap(h http.Handler) http.Handler {
	for i := len(ms) - 1; i >= 0; i-- {
		h = ms[i](h)
	}
	return h
}
