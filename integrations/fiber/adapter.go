package fiber

import (
	"bytes"
	"io"
	"net/http"
	"net/url"

	"github.com/gofiber/fiber/v2"
	"github.com/marshallshelly/beacon-auth/core"
)

// responseAdapter adapts Fiber context to http.ResponseWriter
type responseAdapter struct {
	c          *fiber.Ctx
	statusCode int
	headers    http.Header
	body       *bytes.Buffer
}

// newResponseAdapter creates a new response adapter
func newResponseAdapter(c *fiber.Ctx) *responseAdapter {
	return &responseAdapter{
		c:          c,
		statusCode: http.StatusOK,
		headers:    make(http.Header),
		body:       &bytes.Buffer{},
	}
}

// Header implements http.ResponseWriter
func (w *responseAdapter) Header() http.Header {
	return w.headers
}

// Write implements http.ResponseWriter
func (w *responseAdapter) Write(data []byte) (int, error) {
	return w.body.Write(data)
}

// WriteHeader implements http.ResponseWriter
func (w *responseAdapter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
}

// flush writes the buffered response to Fiber context
func (w *responseAdapter) flush() error {
	// Set headers
	for key, values := range w.headers {
		for _, value := range values {
			w.c.Set(key, value)
		}
	}

	// Set status and body
	w.c.Status(w.statusCode)
	return w.c.Send(w.body.Bytes())
}

// requestAdapter adapts Fiber context to http.Request
type requestAdapter struct {
	*http.Request
	c *fiber.Ctx
}

// newRequestAdapter creates a new request adapter
func newRequestAdapter(c *fiber.Ctx) *requestAdapter {
	// Parse URL from Fiber context
	parsedURL, _ := url.Parse(c.OriginalURL())
	if parsedURL == nil {
		parsedURL = &url.URL{Path: c.Path()}
	}

	// Create standard http.Request from Fiber request
	req := &http.Request{
		Method: c.Method(),
		URL:    parsedURL,
		Header: make(http.Header),
		Body:   io.NopCloser(bytes.NewReader(c.Body())),
		Host:   string(c.Request().Host()),
	}

	// Copy headers
	for key, value := range c.Request().Header.All() {
		req.Header.Add(string(key), string(value))
	}

	// Set RemoteAddr
	req.RemoteAddr = c.IP()

	// Add session and user to context if present
	ctx := req.Context()
	if session := GetSession(c); session != nil {
		ctx = core.WithSession(ctx, session)
	}
	if user := GetUser(c); user != nil {
		ctx = core.WithUser(ctx, user)
	}
	req = req.WithContext(ctx)

	return &requestAdapter{
		Request: req,
		c:       c,
	}
}

// Cookie retrieves a cookie from Fiber context
func (r *requestAdapter) Cookie(name string) (*http.Cookie, error) {
	value := r.c.Cookies(name)
	if value == "" {
		return nil, http.ErrNoCookie
	}

	return &http.Cookie{
		Name:  name,
		Value: value,
	}, nil
}
