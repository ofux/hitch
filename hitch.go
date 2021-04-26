package hitch

import (
	"net/http"
	"strings"

	"github.com/julienschmidt/httprouter"
)

// Middleware wraps an http.Handler, returning a new http.Handler.
type Middleware func(next http.Handler) http.Handler

// Hitch ties httprouter, context, and middleware up in a bow.
type Hitch struct {
	router *httprouter.Router

	basePath    string
	middlewares []Middleware
}

// New initializes a new Hitch.
func New() *Hitch {
	r := httprouter.New()
	r.HandleMethodNotAllowed = false // may cause problems otherwise
	return &Hitch{
		router: r,
	}
}

// Router returns the internal httprouter.Router
func (h *Hitch) Router() *httprouter.Router {
	return h.router
}

// SubPath returns a new Hitch in which a set of sub-routes can be defined. It can be used for inner
// routes that share a common middleware. It inherits all middlewares and base-path of the parent Hitch.
func (h *Hitch) SubPath(path string) *Hitch {
	var middlewaresCopy []Middleware
	if len(h.middlewares) > 0 {
		middlewaresCopy = make([]Middleware, len(h.middlewares))
		copy(middlewaresCopy, h.middlewares)
	}

	return &Hitch{
		router:      h.router,
		basePath:    h.path(path),
		middlewares: middlewaresCopy,
	}
}

// WithMiddleware installs one or more middleware in the Hitch request cycle.
func (h *Hitch) WithMiddleware(middleware ...Middleware) *Hitch {
	newHitch := h.SubPath("")
	newHitch.middlewares = append(newHitch.middlewares, middleware...)
	return newHitch
}

// WithHandlerMiddleware registers an http.Handler as a middleware.
func (h *Hitch) WithHandlerMiddleware(handler http.Handler) *Hitch {
	return h.WithMiddleware(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			handler.ServeHTTP(w, req)
			next.ServeHTTP(w, req)
		})
	})
}

// Handle registers a handler for the given method and path.
func (h *Hitch) Handle(method, path string, handler http.Handler, middleware ...Middleware) {
	for i := len(middleware) - 1; i >= 0; i-- {
		handler = middleware[i](handler)
	}
	for i := len(h.middlewares) - 1; i >= 0; i-- {
		handler = h.middlewares[i](handler)
	}
	h.router.Handler(method, h.path(path), handler)
}

// HandleFunc registers a func handler for the given method and path.
func (h *Hitch) HandleFunc(method, path string, handler http.HandlerFunc, middleware ...Middleware) {
	h.Handle(method, path, handler, middleware...)
}

// GET registers a GET handler for the given path.
func (h *Hitch) GET(path string, handler http.HandlerFunc, middleware ...Middleware) {
	h.Handle(http.MethodGet, path, handler, middleware...)
}

// PUT registers a PUT handler for the given path.
func (h *Hitch) PUT(path string, handler http.HandlerFunc, middleware ...Middleware) {
	h.Handle(http.MethodPut, path, handler, middleware...)
}

// POST registers a POST handler for the given path.
func (h *Hitch) POST(path string, handler http.HandlerFunc, middleware ...Middleware) {
	h.Handle(http.MethodPost, path, handler, middleware...)
}

// PATCH registers a PATCH handler for the given path.
func (h *Hitch) PATCH(path string, handler http.HandlerFunc, middleware ...Middleware) {
	h.Handle(http.MethodPatch, path, handler, middleware...)
}

// DELETE registers a DELETE handler for the given path.
func (h *Hitch) DELETE(path string, handler http.HandlerFunc, middleware ...Middleware) {
	h.Handle(http.MethodDelete, path, handler, middleware...)
}

// OPTIONS registers a OPTIONS handler for the given path.
func (h *Hitch) OPTIONS(path string, handler http.HandlerFunc, middleware ...Middleware) {
	h.Handle(http.MethodOptions, path, handler, middleware...)
}

// Params returns the httprouter.Params for req.
// This is just a pass-through to httprouter.ParamsFromContext.
func Params(req *http.Request) httprouter.Params {
	return httprouter.ParamsFromContext(req.Context())
}

func (h *Hitch) path(p string) string {
	base := strings.TrimSuffix(h.basePath, "/")

	if p != "" && !strings.HasPrefix(p, "/") {
		p = "/" + p
	}

	return base + p
}
