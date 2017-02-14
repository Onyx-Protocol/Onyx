package core

// HealthSetter returns a function that, when called,
// sets the named health status in the map returned by "/health".
// The returned function is safe to call concurrently with ServeHTTP.
func (a *API) HealthSetter(name string) func(error) {
	return func(err error) { a.setHealth(name, err) }
}

func (a *API) setHealth(name string, err error) {
	a.healthMu.Lock()
	defer a.healthMu.Unlock()
	if a.healthErrors == nil {
		a.healthErrors = make(map[string]interface{})
	}
	if err == nil {
		a.healthErrors[name] = nil
	} else {
		a.healthErrors[name] = err.Error() // convert to immutable string
	}
}

func (a *API) health() (x struct {
	Errors map[string]interface{} `json:"errors"`
}) {
	x.Errors = make(map[string]interface{})
	a.healthMu.Lock()
	defer a.healthMu.Unlock()
	for name, s := range a.healthErrors {
		x.Errors[name] = s // copy for safe serialization
	}
	return
}
