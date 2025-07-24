package a1

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
)

// A1PolicyValidatorImpl implements policy validation according to JSON Schema
type A1PolicyValidatorImpl struct {
	logger *logrus.Logger
}

// NewA1PolicyValidator creates a new policy validator
func NewA1PolicyValidator() A1PolicyValidator {
	return &A1PolicyValidatorImpl{
		logger: logrus.WithField("component", "a1-validator").Logger,
	}
}

// ValidatePolicyType validates a policy type definition
func (v *A1PolicyValidatorImpl) ValidatePolicyType(policyType *A1PolicyType) error {
	if policyType.PolicyTypeID == "" {
		return fmt.Errorf("policy type ID cannot be empty")
	}

	if policyType.Name == "" {
		return fmt.Errorf("policy type name cannot be empty")
	}

	if policyType.PolicySchema == nil {
		return fmt.Errorf("policy schema cannot be nil")
	}

	// Validate the schema structure
	if err := v.ValidateSchema(policyType.PolicySchema); err != nil {
		return fmt.Errorf("invalid policy schema: %w", err)
	}

	return nil
}

// ValidatePolicy validates a policy against its policy type schema
func (v *A1PolicyValidatorImpl) ValidatePolicy(policy *A1Policy, policyType *A1PolicyType) error {
	if policy.PolicyID == "" {
		return fmt.Errorf("policy ID cannot be empty")
	}

	if policy.PolicyTypeID != policyType.PolicyTypeID {
		return fmt.Errorf("policy type ID mismatch: expected %s, got %s",
			policyType.PolicyTypeID, policy.PolicyTypeID)
	}

	if policy.PolicyData == nil {
		return fmt.Errorf("policy data cannot be nil")
	}

	// Validate policy data against schema
	if err := v.validateAgainstSchema(policy.PolicyData, policyType.PolicySchema); err != nil {
		return fmt.Errorf("policy data validation failed: %w", err)
	}

	return nil
}

// ValidateSchema validates a JSON schema structure
func (v *A1PolicyValidatorImpl) ValidateSchema(schema map[string]interface{}) error {
	// Check for required JSON Schema fields
	schemaType, exists := schema["type"]
	if !exists {
		return fmt.Errorf("schema must have a 'type' field")
	}

	typeStr, ok := schemaType.(string)
	if !ok {
		return fmt.Errorf("schema 'type' must be a string")
	}

	// Validate based on type
	switch typeStr {
	case "object":
		return v.validateObjectSchema(schema)
	case "array":
		return v.validateArraySchema(schema)
	case "string":
		return v.validateStringSchema(schema)
	case "number", "integer":
		return v.validateNumberSchema(schema)
	case "boolean":
		return nil // Boolean schemas are simple
	default:
		return fmt.Errorf("unsupported schema type: %s", typeStr)
	}
}

// validateAgainstSchema validates data against a JSON schema
func (v *A1PolicyValidatorImpl) validateAgainstSchema(data interface{}, schema map[string]interface{}) error {
	schemaType, exists := schema["type"]
	if !exists {
		return fmt.Errorf("schema missing 'type' field")
	}

	typeStr, ok := schemaType.(string)
	if !ok {
		return fmt.Errorf("schema 'type' must be a string")
	}

	switch typeStr {
	case "object":
		return v.validateObjectData(data, schema)
	case "array":
		return v.validateArrayData(data, schema)
	case "string":
		return v.validateStringData(data, schema)
	case "number":
		return v.validateNumberData(data, schema, false)
	case "integer":
		return v.validateNumberData(data, schema, true)
	case "boolean":
		return v.validateBooleanData(data, schema)
	default:
		return fmt.Errorf("unsupported schema type: %s", typeStr)
	}
}

// Schema validation helpers

func (v *A1PolicyValidatorImpl) validateObjectSchema(schema map[string]interface{}) error {
	// Check properties if they exist
	if properties, exists := schema["properties"]; exists {
		propsMap, ok := properties.(map[string]interface{})
		if !ok {
			return fmt.Errorf("'properties' must be an object")
		}

		// Validate each property schema
		for propName, propSchema := range propsMap {
			propSchemaMap, ok := propSchema.(map[string]interface{})
			if !ok {
				return fmt.Errorf("property '%s' schema must be an object", propName)
			}

			if err := v.ValidateSchema(propSchemaMap); err != nil {
				return fmt.Errorf("invalid schema for property '%s': %w", propName, err)
			}
		}
	}

	// Validate required array if it exists
	if required, exists := schema["required"]; exists {
		reqArray, ok := required.([]interface{})
		if !ok {
			// Try []string
			if reqStringArray, ok := required.([]string); ok {
				// Convert to []interface{}
				reqArray = make([]interface{}, len(reqStringArray))
				for i, s := range reqStringArray {
					reqArray[i] = s
				}
			} else {
				return fmt.Errorf("'required' must be an array")
			}
		}

		// Validate that all required fields are strings
		for i, req := range reqArray {
			if _, ok := req.(string); !ok {
				return fmt.Errorf("required field %d must be a string", i)
			}
		}
	}

	return nil
}

func (v *A1PolicyValidatorImpl) validateArraySchema(schema map[string]interface{}) error {
	// Check items if they exist
	if items, exists := schema["items"]; exists {
		itemsMap, ok := items.(map[string]interface{})
		if !ok {
			return fmt.Errorf("'items' must be an object")
		}

		if err := v.ValidateSchema(itemsMap); err != nil {
			return fmt.Errorf("invalid items schema: %w", err)
		}
	}

	return nil
}

func (v *A1PolicyValidatorImpl) validateStringSchema(schema map[string]interface{}) error {
	// Validate enum if it exists
	if enum, exists := schema["enum"]; exists {
		enumArray, ok := enum.([]interface{})
		if !ok {
			return fmt.Errorf("'enum' must be an array")
		}

		// All enum values should be strings for string type
		for i, val := range enumArray {
			if _, ok := val.(string); !ok {
				return fmt.Errorf("enum value %d must be a string", i)
			}
		}
	}

	return nil
}

func (v *A1PolicyValidatorImpl) validateNumberSchema(schema map[string]interface{}) error {
	// Validate minimum if it exists
	if minimum, exists := schema["minimum"]; exists {
		if _, ok := minimum.(float64); !ok {
			if _, ok := minimum.(int); !ok {
				return fmt.Errorf("'minimum' must be a number")
			}
		}
	}

	// Validate maximum if it exists
	if maximum, exists := schema["maximum"]; exists {
		if _, ok := maximum.(float64); !ok {
			if _, ok := maximum.(int); !ok {
				return fmt.Errorf("'maximum' must be a number")
			}
		}
	}

	return nil
}

// Data validation helpers

func (v *A1PolicyValidatorImpl) validateObjectData(data interface{}, schema map[string]interface{}) error {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("expected object, got %T", data)
	}

	// Check required fields
	if required, exists := schema["required"]; exists {
		reqArray, ok := required.([]interface{})
		if !ok {
			// Try []string
			if reqStringArray, ok := required.([]string); ok {
				reqArray = make([]interface{}, len(reqStringArray))
				for i, s := range reqStringArray {
					reqArray[i] = s
				}
			}
		}

		if reqArray != nil {
			for _, req := range reqArray {
				reqStr, ok := req.(string)
				if !ok {
					continue
				}

				if _, exists := dataMap[reqStr]; !exists {
					return fmt.Errorf("missing required field: %s", reqStr)
				}
			}
		}
	}

	// Validate properties
	if properties, exists := schema["properties"]; exists {
		propsMap, ok := properties.(map[string]interface{})
		if !ok {
			return fmt.Errorf("properties schema must be an object")
		}

		for fieldName, fieldValue := range dataMap {
			if propSchema, exists := propsMap[fieldName]; exists {
				propSchemaMap, ok := propSchema.(map[string]interface{})
				if !ok {
					return fmt.Errorf("property schema for '%s' must be an object", fieldName)
				}

				if err := v.validateAgainstSchema(fieldValue, propSchemaMap); err != nil {
					return fmt.Errorf("validation failed for field '%s': %w", fieldName, err)
				}
			} else {
				// Check if additional properties are allowed
				if additionalProps, exists := schema["additionalProperties"]; exists {
					if allowed, ok := additionalProps.(bool); ok && !allowed {
						return fmt.Errorf("additional property '%s' not allowed", fieldName)
					}
					// If additionalProperties is a schema, validate against it
					if addPropsSchema, ok := additionalProps.(map[string]interface{}); ok {
						if err := v.validateAgainstSchema(fieldValue, addPropsSchema); err != nil {
							return fmt.Errorf("validation failed for additional property '%s': %w", fieldName, err)
						}
					}
				}
			}
		}
	}

	return nil
}

func (v *A1PolicyValidatorImpl) validateArrayData(data interface{}, schema map[string]interface{}) error {
	dataArray, ok := data.([]interface{})
	if !ok {
		return fmt.Errorf("expected array, got %T", data)
	}

	// Validate items if schema specifies
	if items, exists := schema["items"]; exists {
		itemsMap, ok := items.(map[string]interface{})
		if !ok {
			return fmt.Errorf("items schema must be an object")
		}

		for i, item := range dataArray {
			if err := v.validateAgainstSchema(item, itemsMap); err != nil {
				return fmt.Errorf("validation failed for item %d: %w", i, err)
			}
		}
	}

	return nil
}

func (v *A1PolicyValidatorImpl) validateStringData(data interface{}, schema map[string]interface{}) error {
	dataStr, ok := data.(string)
	if !ok {
		return fmt.Errorf("expected string, got %T", data)
	}

	// Check enum constraint
	if enum, exists := schema["enum"]; exists {
		enumArray, ok := enum.([]interface{})
		if !ok {
			return fmt.Errorf("enum must be an array")
		}

		found := false
		for _, val := range enumArray {
			if valStr, ok := val.(string); ok && valStr == dataStr {
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("value '%s' not allowed by enum constraint", dataStr)
		}
	}

	// Check minLength
	if minLength, exists := schema["minLength"]; exists {
		if minLen, ok := minLength.(float64); ok {
			if len(dataStr) < int(minLen) {
				return fmt.Errorf("string length %d is less than minimum %d", len(dataStr), int(minLen))
			}
		}
	}

	// Check maxLength
	if maxLength, exists := schema["maxLength"]; exists {
		if maxLen, ok := maxLength.(float64); ok {
			if len(dataStr) > int(maxLen) {
				return fmt.Errorf("string length %d exceeds maximum %d", len(dataStr), int(maxLen))
			}
		}
	}

	// Check pattern (simplified - would use regex in production)
	if pattern, exists := schema["pattern"]; exists {
		if patternStr, ok := pattern.(string); ok {
			// Simplified pattern matching
			if strings.Contains(patternStr, "^") && !strings.HasPrefix(dataStr, strings.TrimPrefix(patternStr, "^")) {
				return fmt.Errorf("string does not match pattern: %s", patternStr)
			}
		}
	}

	return nil
}

func (v *A1PolicyValidatorImpl) validateNumberData(data interface{}, schema map[string]interface{}, mustBeInteger bool) error {
	var numVal float64
	var ok bool

	if mustBeInteger {
		// For integer type, accept both int and float64 but ensure it's a whole number
		switch val := data.(type) {
		case int:
			numVal = float64(val)
			ok = true
		case float64:
			numVal = val
			ok = true
			// Check if it's actually a whole number
			if numVal != float64(int(numVal)) {
				return fmt.Errorf("expected integer, got float with decimal part")
			}
		case json.Number:
			if f, err := val.Float64(); err == nil {
				numVal = f
				ok = true
				if mustBeInteger && numVal != float64(int(numVal)) {
					return fmt.Errorf("expected integer, got float with decimal part")
				}
			}
		}
	} else {
		// For number type, accept int, float64, and json.Number
		switch val := data.(type) {
		case int:
			numVal = float64(val)
			ok = true
		case float64:
			numVal = val
			ok = true
		case json.Number:
			if f, err := val.Float64(); err == nil {
				numVal = f
				ok = true
			}
		}
	}

	if !ok {
		expectedType := "number"
		if mustBeInteger {
			expectedType = "integer"
		}
		return fmt.Errorf("expected %s, got %T", expectedType, data)
	}

	// Check minimum constraint
	if minimum, exists := schema["minimum"]; exists {
		var minVal float64
		switch min := minimum.(type) {
		case int:
			minVal = float64(min)
		case float64:
			minVal = min
		case json.Number:
			if f, err := min.Float64(); err == nil {
				minVal = f
			}
		}

		if numVal < minVal {
			return fmt.Errorf("value %v is less than minimum %v", numVal, minVal)
		}
	}

	// Check maximum constraint
	if maximum, exists := schema["maximum"]; exists {
		var maxVal float64
		switch max := maximum.(type) {
		case int:
			maxVal = float64(max)
		case float64:
			maxVal = max
		case json.Number:
			if f, err := max.Float64(); err == nil {
				maxVal = f
			}
		}

		if numVal > maxVal {
			return fmt.Errorf("value %v exceeds maximum %v", numVal, maxVal)
		}
	}

	return nil
}

func (v *A1PolicyValidatorImpl) validateBooleanData(data interface{}, schema map[string]interface{}) error {
	if _, ok := data.(bool); !ok {
		return fmt.Errorf("expected boolean, got %T", data)
	}

	return nil
}

// ValidateSpecificPolicyTypes provides validation for O-RAN specific policy types

// ValidateQoSPolicy validates a QoS policy according to O-RAN specifications
func (v *A1PolicyValidatorImpl) ValidateQoSPolicy(policyData map[string]interface{}) error {
	// Validate QCI
	qci, exists := policyData["qci"]
	if !exists {
		return fmt.Errorf("QCI is required")
	}

	qciNum, ok := qci.(float64)
	if !ok {
		if qciInt, ok := qci.(int); ok {
			qciNum = float64(qciInt)
		} else {
			return fmt.Errorf("QCI must be a number")
		}
	}

	if qciNum < 1 || qciNum > 9 {
		return fmt.Errorf("QCI must be between 1 and 9")
	}

	// Validate priority level
	if priority, exists := policyData["priority_level"]; exists {
		priorityNum, ok := priority.(float64)
		if !ok {
			if priorityInt, ok := priority.(int); ok {
				priorityNum = float64(priorityInt)
			} else {
				return fmt.Errorf("priority_level must be a number")
			}
		}

		if priorityNum < 1 || priorityNum > 15 {
			return fmt.Errorf("priority_level must be between 1 and 15")
		}
	}

	// Validate packet delay budget
	if pdb, exists := policyData["packet_delay_budget"]; exists {
		pdbNum, ok := pdb.(float64)
		if !ok {
			if pdbInt, ok := pdb.(int); ok {
				pdbNum = float64(pdbInt)
			} else {
				return fmt.Errorf("packet_delay_budget must be a number")
			}
		}

		if pdbNum < 50 || pdbNum > 300 {
			return fmt.Errorf("packet_delay_budget must be between 50 and 300 ms")
		}
	}

	return nil
}

// ValidateRRMPolicy validates a Radio Resource Management policy
func (v *A1PolicyValidatorImpl) ValidateRRMPolicy(policyData map[string]interface{}) error {
	// Validate resource type
	resourceType, exists := policyData["resource_type"]
	if !exists {
		return fmt.Errorf("resource_type is required")
	}

	resourceTypeStr, ok := resourceType.(string)
	if !ok {
		return fmt.Errorf("resource_type must be a string")
	}

	validResourceTypes := map[string]bool{
		"PRB":      true,
		"SPECTRUM": true,
		"POWER":    true,
	}

	if !validResourceTypes[resourceTypeStr] {
		return fmt.Errorf("invalid resource_type: %s", resourceTypeStr)
	}

	// Validate allocation strategy
	strategy, exists := policyData["allocation_strategy"]
	if !exists {
		return fmt.Errorf("allocation_strategy is required")
	}

	strategyStr, ok := strategy.(string)
	if !ok {
		return fmt.Errorf("allocation_strategy must be a string")
	}

	validStrategies := map[string]bool{
		"ROUND_ROBIN":       true,
		"PROPORTIONAL_FAIR": true,
		"MAX_THROUGHPUT":    true,
	}

	if !validStrategies[strategyStr] {
		return fmt.Errorf("invalid allocation_strategy: %s", strategyStr)
	}

	return nil
}
