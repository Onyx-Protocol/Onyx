package core

// HealthSetter returns a function that, when called,
// sets the named health status in the map returned by "/health".
// The returned function is safe to call concurrently with ServeHTTP.
func (h *Handler) HealthSetter(name string) func(error) {
	return func(err error) { h.setHealth(name, err) }
}

func (h *Handler) setHealth(name string, err error) {
	h.healthMu.Lock()
	defer h.healthMu.Unlock()
	if h.healthErrors == nil {
		h.healthErrors = make(map[string]interface{})
	}
	if err == nil {
		h.healthErrors[name] = nil
	} else {
		h.healthErrors[name] = err.Error() // convert to immutable string
	}
}

func (h *Handler) health() (x struct {
	Errors map[string]interface{} `json:"errors"`
}) {
	x.Errors = make(map[string]interface{})
	h.healthMu.Lock()
	defer h.healthMu.Unlock()
	for name, s := range h.healthErrors {
		x.Errors[name] = s // copy for safe serialization
	}
	return
}
