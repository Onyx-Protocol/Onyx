package state

// BackedView is a View to the state backed by another View.
// It is used by txpool's view and various transient views
// during validation process.
type BackedView struct {
	// Frontend is used to fetch, store and cache data (if Cache=true).
	// If data is missing in frontend, it is fetched from the backend view.
	Frontend View

	// Backend view is accessed when the data is not found
	// in the frontend view.
	Backend View

	// Cache indicates if any fallback to backend must lead to
	// storing loaded data in the frontend view.
	Cache bool
}
