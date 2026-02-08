package api

import (
	"fmt"
	"strings"
)

// validateEndpoint validates that the endpoint name exists
func (a *Api) validateEndpoint(value any) error {
	name, ok := value.(string)
	if !ok {
		return fmt.Errorf("endpoint must be a string")
	}

	for _, endpoint := range a.endpoints {
		if endpoint.Name == name {
			return nil
		}
	}

	return fmt.Errorf("unknown endpoint: %s. Available: %s",
		name, strings.Join(a.getEndpointNames(), ", "))
}

// getEndpointNames returns a list of all endpoint names
func (a *Api) getEndpointNames() []string {
	names := make([]string, len(a.endpoints))
	for i, endpoint := range a.endpoints {
		names[i] = endpoint.Name
	}
	return names
}

// findEndpoint finds an endpoint by name
func (a *Api) findEndpoint(name string) *Endpoint {
	for i := range a.endpoints {
		if a.endpoints[i].Name == name {
			return &a.endpoints[i]
		}
	}
	return nil
}

// Validation Function Registry
// Similar to hooks, endpoint validators can be registered by name
// This allows YAML configuration to reference validators without direct function pointers

type validatorRegistry struct {
	validators map[string]EndpointValidator
}

var defaultValidatorRegistry = &validatorRegistry{
	validators: make(map[string]EndpointValidator),
}

// RegisterValidator registers an endpoint validator with a given name
func RegisterValidator(name string, validator EndpointValidator) {
	defaultValidatorRegistry.validators[name] = validator
}

// GetValidator retrieves a registered validator by name
// Returns nil if the validator is not found
func GetValidator(name string) EndpointValidator {
	if validator, ok := defaultValidatorRegistry.validators[name]; ok {
		return validator
	}
	return nil
}

// ValidatePositiveIntParam ensures specific params are positive integers
func ValidatePositiveIntParam(paramNames ...string) EndpointValidator {
	return func(params EndpointValidationParams) error {
		for _, paramName := range paramNames {
			// Check URL params
			if val, ok := params.URLParams[paramName]; ok {
				if num, ok := val.(float64); ok {
					if num <= 0 {
						return fmt.Errorf("%s must be a positive integer, got: %v", paramName, num)
					}
				}
			}
			// Check query params
			if val, ok := params.QueryParams[paramName]; ok {
				if num, ok := val.(float64); ok {
					if num <= 0 {
						return fmt.Errorf("%s must be a positive integer, got: %v", paramName, num)
					}
				}
			}
		}
		return nil
	}
}

// ValidateRequiredParams ensures specific params are present and non-empty
func ValidateRequiredParams(paramNames ...string) EndpointValidator {
	return func(params EndpointValidationParams) error {
		for _, paramName := range paramNames {
			found := false

			// Check URL params
			if val, ok := params.URLParams[paramName]; ok && val != "" {
				found = true
			}
			// Check query params
			if val, ok := params.QueryParams[paramName]; ok && val != "" {
				found = true
			}

			if !found {
				return fmt.Errorf("required parameter missing or empty: %s", paramName)
			}
		}
		return nil
	}
}

// ValidateBodyMaxSize ensures request body doesn't exceed size limit
func ValidateBodyMaxSize(maxBytes int) EndpointValidator {
	return func(params EndpointValidationParams) error {
		if len(params.Body) > maxBytes {
			return fmt.Errorf("request body exceeds maximum size of %d bytes", maxBytes)
		}
		return nil
	}
}
