package core

// healthSetter returns a function that, when called,
// sets the named health status in the map returned by "/health".
// The returned function is safe to call concurrently with ServeHTTP.
func (a *API) healthSetter(name string) func(error) {
	return func(err error) { a.setHealth(name, err) }
}

func (a *API) setHealth(name string, err error) {
	a.healthMu.Lock()
	defer a.healthMu.Unlock()
	if a.healthErrors == nil {
		a.healthErrors = make(map[string]string)
	}
	if err == nil {
		delete(a.healthErrors, name)
	} else {
		a.healthErrors[name] = err.Error() // convert to immutable string
	}
}

func (a *API) health() (x struct {
	Errors map[string]string `json:"errors"`
}) {
	x.Errors = make(map[string]string)

	if err := a.sdb.RaftService().Err(); err != nil {
		x.Errors["raft"] = err.Error()
	}
	if err := a.options.Err(); err != nil {
		x.Errors["config"] = err.Error()
	}

	a.healthMu.Lock()
	defer a.healthMu.Unlock()
	for name, s := range a.healthErrors {
		x.Errors[name] = s // copy for safe serialization
	}
	return
}
