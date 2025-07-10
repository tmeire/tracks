package tracks

import "net/http"

type MiddlewareBuilder func(r Router) Middleware

type Middleware func(h http.Handler) (http.Handler, error)

type middlewares struct {
	l []Middleware
}

func (ms *middlewares) Apply(m Middleware) {
	if m != nil {
		ms.l = append(ms.l, m)
	}
}

func (ms *middlewares) Wrap(r Router, h http.Handler, mws ...MiddlewareBuilder) (http.Handler, error) {
	var err error
	// First, apply any additional middlewares
	for i := len(mws) - 1; i >= 0; i-- {
		h, err = mws[i](r)(h)
		if err != nil {
			return nil, err
		}
	}
	// Then, apply middlewares from the list
	for i := len(ms.l) - 1; i >= 0; i-- {
		h, err = ms.l[i](h)
		if err != nil {
			return nil, err
		}
	}
	return h, nil
}
