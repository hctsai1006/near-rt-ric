package o1

import (
	"fmt"
	"strings"
	"time"
)

// YANG model utilities

// ValidateYANGModel validates a YANG model definition
func ValidateYANGModel(model *YANGModel) error {
	if model.Name == "" {
		return fmt.Errorf("YANG model name is required")
	}
	if model.Namespace == "" {
		return fmt.Errorf("YANG model namespace is required")
	}
	if model.Version == "" {
		return fmt.Errorf("YANG model version is required")
	}
	if model.Revision == "" {
		return fmt.Errorf("YANG model revision is required")
	}
	return nil
}

// NETCONF utility functions

// CreateNetconfHello creates a NETCONF hello message
func CreateNetconfHello(capabilities []NetconfCapability, sessionID uint32) map[string]interface{} {
	return map[string]interface{}{
		"hello": map[string]interface{}{
			"xmlns":      NetconfBase10,
			"session-id": sessionID,
			"capabilities": map[string]interface{}{
				"capability": capabilitiesToStrings(capabilities),
			},
		},
	}
}

// CreateNetconfRPCReply creates a NETCONF RPC reply message
func CreateNetconfRPCReply(messageID string, data interface{}) map[string]interface{} {
	reply := map[string]interface{}{
		"rpc-reply": map[string]interface{}{
			"xmlns":      NetconfBase10,
			"message-id": messageID,
		},
	}

	if data != nil {
		reply["rpc-reply"].(map[string]interface{})["data"] = data
	} else {
		reply["rpc-reply"].(map[string]interface{})["ok"] = struct{}{}
	}

	return reply
}

// CreateNetconfRPCError creates a NETCONF RPC error message
func CreateNetconfRPCError(messageID, errorType, errorTag, severity, message string) map[string]interface{} {
	return map[string]interface{}{
		"rpc-reply": map[string]interface{}{
			"xmlns":      NetconfBase10,
			"message-id": messageID,
			"rpc-error": map[string]interface{}{
				"error-type":     errorType,
				"error-tag":      errorTag,
				"error-severity": severity,
				"error-message":  message,
			},
		},
	}
}

// capabilitiesToStrings converts capabilities to string slice
func capabilitiesToStrings(capabilities []NetconfCapability) []string {
	result := make([]string, len(capabilities))
	for i, cap := range capabilities {
		result[i] = cap.URI
	}
	return result
}

// Alarm utility functions

// FormatAlarmSeverity formats alarm severity for display
func FormatAlarmSeverity(severity AlarmSeverity) string {
	switch severity {
	case AlarmCritical:
		return "CRITICAL"
	case AlarmMajor:
		return "MAJOR"
	case AlarmMinor:
		return "MINOR"
	case AlarmWarning:
		return "WARNING"
	default:
		return "UNKNOWN"
	}
}

// GetAlarmSeverityLevel returns numeric level for severity comparison
func GetAlarmSeverityLevel(severity AlarmSeverity) int {
	switch severity {
	case AlarmCritical:
		return 4
	case AlarmMajor:
		return 3
	case AlarmMinor:
		return 2
	case AlarmWarning:
		return 1
	default:
		return -1
	}
}

// Performance metric utilities

// CalculateMetricGrowthRate calculates growth rate between two metric values
func CalculateMetricGrowthRate(oldValue, newValue float64, timeDiff float64) float64 {
	if timeDiff <= 0 || oldValue == 0 {
		return 0
	}
	return ((newValue - oldValue) / oldValue) * 100 / timeDiff
}

// FormatMetricValue formats metric value based on type and unit
func FormatMetricValue(value interface{}, unit string) string {
	switch v := value.(type) {
	case float64:
		if unit == "%" {
			return fmt.Sprintf("%.2f%%", v)
		} else if unit == "bytes" {
			return formatBytes(int64(v))
		} else {
			return fmt.Sprintf("%.2f %s", v, unit)
		}
	case int64:
		if unit == "bytes" {
			return formatBytes(v)
		} else {
			return fmt.Sprintf("%d %s", v, unit)
		}
	case int:
		return fmt.Sprintf("%d %s", v, unit)
	default:
		return fmt.Sprintf("%v %s", v, unit)
	}
}

// formatBytes formats byte values with appropriate units
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	units := []string{"KB", "MB", "GB", "TB", "PB"}
	return fmt.Sprintf("%.1f %s", float64(bytes)/float64(div), units[exp])
}

// Configuration utilities

// ValidateDatastore validates datastore type
func ValidateDatastore(datastore DatastoreType) error {
	if !datastore.IsValid() {
		return fmt.Errorf("invalid datastore type: %s", datastore)
	}
	return nil
}

// ValidateNetconfOperation validates NETCONF operation
func ValidateNetconfOperation(operation NetconfOperation) error {
	if !operation.IsValid() {
		return fmt.Errorf("invalid NETCONF operation: %s", operation)
	}
	return nil
}

// Security utilities

// ValidateSecurityRule validates a security rule
func ValidateSecurityRule(rule *SecurityRule) error {
	if rule.RuleID == "" {
		return fmt.Errorf("security rule ID is required")
	}
	if rule.RuleName == "" {
		return fmt.Errorf("security rule name is required")
	}
	if rule.Action == "" {
		return fmt.Errorf("security rule action is required")
	}

	validActions := map[string]bool{"allow": true, "deny": true, "log": true}
	if !validActions[rule.Action] {
		return fmt.Errorf("invalid security rule action: %s", rule.Action)
	}

	return nil
}

// ValidateCertificate validates certificate data
func ValidateCertificate(cert *Certificate) error {
	if cert.CertificateID == "" {
		return fmt.Errorf("certificate ID is required")
	}
	if cert.CommonName == "" {
		return fmt.Errorf("certificate common name is required")
	}
	if cert.Subject == "" {
		return fmt.Errorf("certificate subject is required")
	}
	if cert.Issuer == "" {
		return fmt.Errorf("certificate issuer is required")
	}
	if cert.SerialNumber == "" {
		return fmt.Errorf("certificate serial number is required")
	}
	return nil
}

// Software management utilities

// ValidateSoftwareVersion validates software version information
func ValidateSoftwareVersion(software *SoftwareVersion) error {
	if software.Name == "" {
		return fmt.Errorf("software name is required")
	}
	if software.Version == "" {
		return fmt.Errorf("software version is required")
	}

	validStatuses := map[string]bool{"active": true, "inactive": true, "corrupted": true}
	if !validStatuses[software.Status] {
		return fmt.Errorf("invalid software status: %s", software.Status)
	}

	return nil
}

// ValidateFileTransfer validates file transfer request
func ValidateFileTransfer(transfer *FileTransfer) error {
	if transfer.TransferID == "" {
		return fmt.Errorf("transfer ID is required")
	}
	if transfer.Operation == "" {
		return fmt.Errorf("transfer operation is required")
	}

	validOperations := map[string]bool{"upload": true, "download": true}
	if !validOperations[transfer.Operation] {
		return fmt.Errorf("invalid transfer operation: %s", transfer.Operation)
	}

	if transfer.LocalPath == "" {
		return fmt.Errorf("local path is required")
	}
	if transfer.RemotePath == "" {
		return fmt.Errorf("remote path is required")
	}
	if transfer.Protocol == "" {
		return fmt.Errorf("transfer protocol is required")
	}

	validProtocols := map[string]bool{"sftp": true, "scp": true, "http": true, "https": true}
	if !validProtocols[transfer.Protocol] {
		return fmt.Errorf("invalid transfer protocol: %s", transfer.Protocol)
	}

	return nil
}

// Event utilities

// CreateO1Event creates an O1 event
func CreateO1Event(eventType, managedObject, source, severity, description string, details map[string]interface{}) *O1Event {
	return &O1Event{
		EventID:       fmt.Sprintf("evt-%d", generateEventID()),
		EventType:     eventType,
		ManagedObject: managedObject,
		Timestamp:     generateTimestamp(),
		Severity:      severity,
		Source:        source,
		Description:   description,
		Details:       details,
	}
}

// generateEventID generates a unique event ID
func generateEventID() int64 {
	return time.Now().UnixNano()
}

// generateTimestamp generates current timestamp
func generateTimestamp() time.Time {
	return time.Now().UTC()
}

// XML utilities for NETCONF message processing

// EscapeXMLContent escapes XML special characters
func EscapeXMLContent(content string) string {
	content = strings.ReplaceAll(content, "&", "&amp;")
	content = strings.ReplaceAll(content, "<", "&lt;")
	content = strings.ReplaceAll(content, ">", "&gt;")
	content = strings.ReplaceAll(content, "\"", "&quot;")
	content = strings.ReplaceAll(content, "'", "&apos;")
	return content
}

// UnescapeXMLContent unescapes XML special characters
func UnescapeXMLContent(content string) string {
	content = strings.ReplaceAll(content, "&amp;", "&")
	content = strings.ReplaceAll(content, "&lt;", "<")
	content = strings.ReplaceAll(content, "&gt;", ">")
	content = strings.ReplaceAll(content, "&quot;", "\"")
	content = strings.ReplaceAll(content, "&apos;", "'")
	return content
}

// ParseXMLPath parses an XPath-like string for configuration filtering
func ParseXMLPath(path string) []string {
	if path == "" {
		return nil
	}

	// Simple path parsing - in production, use proper XPath parser
	parts := strings.Split(strings.Trim(path, "/"), "/")
	var result []string
	for _, part := range parts {
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}
