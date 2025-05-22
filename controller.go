package tracks

import "net/http"

type Controller interface {
	Index(r *http.Request) (any, error)
}

type BaseController struct {
	Router Router
}

func (bc *BaseController) Inject(router Router) {
	bc.Router = router
}

func (bc BaseController) Scheme() string {
	if bc.Router.Secure() {
		return "https"
	}
	return "http"
}
