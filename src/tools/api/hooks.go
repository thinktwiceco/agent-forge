package api

// ApiCallHook is a function that modifies headers before making an API call
// It receives the URL, current headers, and body, and returns modified headers or an error
type ApiCallHook func(url string, headers map[string]string, body string) (map[string]string, error)

// hookRegistry holds registered API call hooks by name
var hookRegistry = make(map[string]ApiCallHook)

// RegisterHook registers an API call hook with a given name
// This allows hooks to be referenced by name in YAML configuration
func RegisterHook(name string, hook ApiCallHook) {
	hookRegistry[name] = hook
}

// GetHook retrieves a registered hook by name
// Returns nil if the hook is not found
func GetHook(name string) ApiCallHook {
	if hook, ok := hookRegistry[name]; ok {
		return hook
	}
	return defaultHook
}

// defaultHook is the default hook that returns headers unchanged
func defaultHook(url string, headers map[string]string, body string) (map[string]string, error) {
	return headers, nil
}
