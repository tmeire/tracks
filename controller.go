package tracks

import "net/http"

type Controller interface {
	Index(r *http.Request) (any, error)
}

type BaseController struct {
	router Router
}

func (bc *BaseController) Inject(router Router) {
	bc.router = router
}

func (bc BaseController) Scheme() string {
	if bc.router.Secure() {
		return "https"
	}
	return "http"
}
