package testutil

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// RedactOptions controls what gets redacted in simulations.
type RedactOptions struct {
	// Domain to replace real TP domain with.
	ReplacementDomain string
	// RealDomain is the actual domain to find and replace.
	RealDomain string
}

// DefaultRedactOptions returns standard redaction settings.
func DefaultRedactOptions(realDomain string) RedactOptions {
	return RedactOptions{
		ReplacementDomain: "test.tpondemand.com",
		RealDomain:        realDomain,
	}
}

// RedactSimulation sanitizes all sensitive data in a simulation.
func RedactSimulation(sim *Simulation, opts RedactOptions) {
	for i := range sim.Pairs {
		redactPair(&sim.Pairs[i], opts)
	}
}

func redactPair(pair *Pair, opts RedactOptions) {
	// Redact query params
	for key, val := range pair.Request.Query {
		if key == "access_token" {
			pair.Request.Query[key] = "REDACTED"
		} else {
			pair.Request.Query[key] = replaceDomain(val, opts)
		}
	}

	// Redact response body
	pair.Response.Body = redactBody(pair.Response.Body, opts)

	// Redact response headers
	for key, val := range pair.Response.Headers {
		pair.Response.Headers[key] = replaceDomain(val, opts)
	}
}

func replaceDomain(s string, opts RedactOptions) string {
	if opts.RealDomain == "" {
		return s
	}
	return strings.ReplaceAll(s, opts.RealDomain, opts.ReplacementDomain)
}

// emailRegex matches email-like patterns.
var emailRegex = regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)

// redactBody sanitizes a JSON response body.
func redactBody(body json.RawMessage, opts RedactOptions) json.RawMessage {
	// Try to parse as JSON object/array
	var parsed any
	if err := json.Unmarshal(body, &parsed); err != nil {
		// Not JSON (e.g., XML string) — do string-level redaction
		var s string
		if err := json.Unmarshal(body, &s); err == nil {
			s = replaceDomain(s, opts)
			s = redactXMLContent(s)
			if result, mErr := json.Marshal(s); mErr == nil {
				return result
			}
		}
		return body
	}

	redacted := redactValue(parsed, opts)
	result, err := json.Marshal(redacted)
	if err != nil {
		return body
	}
	return result
}

// sensitiveFieldPatterns maps field name patterns to redaction strategies.
var sensitiveFieldPatterns = map[string]string{
	"Description":  "text",
	"Login":        "login",
	"Email":        "email",
	"FirstName":    "firstname",
	"LastName":     "lastname",
	"FullName":     "fullname",
	"Icon":         "url",
	"AvatarUri":    "url",
	"Company":      "text",
	"Phone":        "text",
	"Tags":         "text",
	"CustomField1": "text",
	"CustomField2": "text",
	"CustomField3": "text",
}

// entityNameCounters tracks redaction counters for consistent naming.
var entityNameCounters = struct {
	entities map[string]int
}{
	entities: make(map[string]int),
}

func redactValue(v any, opts RedactOptions) any {
	switch val := v.(type) {
	case map[string]any:
		return redactObject(val, opts)
	case []any:
		result := make([]any, len(val))
		for i, item := range val {
			result[i] = redactValue(item, opts)
		}
		return result
	case string:
		return redactString(val, opts)
	default:
		return v
	}
}

func redactObject(obj map[string]any, opts RedactOptions) map[string]any {
	result := make(map[string]any, len(obj))

	// Determine resource type for context-aware redaction
	var resourceType string
	if rt, ok := obj["ResourceType"].(string); ok {
		resourceType = rt
	} else if rt, ok := obj["resourceType"].(string); ok {
		resourceType = rt
	}

	for key, val := range obj {
		result[key] = redactFieldValue(key, val, opts, resourceType)
	}
	return result
}

func redactFieldValue(fieldName string, val any, opts RedactOptions, resourceType string) any {
	// Check if this field should be redacted
	if strategy, ok := sensitiveFieldPatterns[fieldName]; ok {
		if s, ok := val.(string); ok && s != "" {
			return applyRedactionStrategy(strategy, s, opts)
		}
	}

	// Redact "Name" on entities (but keep state/type names)
	if fieldName == "Name" || fieldName == "name" {
		if s, ok := val.(string); ok && s != "" {
			if isEntityName(resourceType) {
				return redactEntityName(resourceType)
			}
		}
	}

	// Redact custom field values (the "Value" field inside CustomFields entries)
	if fieldName == "Value" && resourceType == "" {
		if s, ok := val.(string); ok && s != "" {
			return "Redacted value"
		}
	}

	// Recurse into nested structures
	return redactValue(val, opts)
}

// isEntityName returns true if the resource type should have its Name redacted.
// Only purely structural/classification types keep their real names.
func isEntityName(resourceType string) bool {
	preserveTypes := map[string]bool{
		"EntityState": true,
		"EntityType":  true,
		"Priority":    true,
		"Role":        true,
		"Process":     true,
		"Workflow":    true,
	}
	if keep, found := preserveTypes[resourceType]; found {
		return !keep
	}
	// Unknown or empty type, plus Project, Team, Feature, etc. → redact
	return true
}

func redactEntityName(resourceType string) string {
	if resourceType == "" {
		resourceType = "Entity"
	}
	entityNameCounters.entities[resourceType]++
	return fmt.Sprintf("Test %s %d", resourceType, entityNameCounters.entities[resourceType])
}

func applyRedactionStrategy(strategy, val string, opts RedactOptions) string {
	switch strategy {
	case "email":
		return "testuser@example.com"
	case "login":
		return "testuser"
	case "firstname":
		return "Test"
	case "lastname":
		return "User"
	case "fullname":
		return "Test User"
	case "url":
		return replaceDomain(val, opts)
	case "text":
		return "Redacted text"
	default:
		return val
	}
}

// redactString applies string-level redaction.
func redactString(s string, opts RedactOptions) string {
	s = replaceDomain(s, opts)
	s = emailRegex.ReplaceAllString(s, "testuser@example.com")
	return s
}

// redactXMLContent sanitizes XML content (used for metadata responses).
// Metadata XML contains structural info (type names, field names) which we keep.
// It may also contain Description attributes with real content.
func redactXMLContent(xml string) string {
	// Replace description attributes that might contain sensitive content
	descRegex := regexp.MustCompile(`Description="[^"]*"`)
	xml = descRegex.ReplaceAllString(xml, `Description="Redacted description"`)
	return xml
}

// ResetCounters resets the redaction counters (call between capture runs).
func ResetCounters() {
	entityNameCounters.entities = make(map[string]int)
}
