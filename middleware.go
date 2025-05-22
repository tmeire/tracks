package tracks

import "net/http"

type Middleware func(h http.Handler) http.Handler

type middlewares struct {
	l []Middleware
}

func (ms *middlewares) Apply(m Middleware) {
	if m != nil {
		ms.l = append(ms.l, m)
	}
}

func (ms *middlewares) Wrap(h http.Handler) http.Handler {
	for i := len(ms.l) - 1; i >= 0; i-- {
		h = ms.l[i](h)
	}
	return h
}
