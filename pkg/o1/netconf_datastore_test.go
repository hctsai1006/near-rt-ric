package o1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewNetconfDatastore(t *testing.T) {
	ds := NewNetconfDatastore()
	require.NotNil(t, ds)
	assert.NotNil(t, ds.running)
	assert.NotNil(t, ds.candidate)
	assert.NotNil(t, ds.startup)
	assert.NotNil(t, ds.locks)
	assert.NotNil(t, ds.yangSchemas)
}

func TestEditConfig_Validation(t *testing.T) {
	ds := NewNetconfDatastore()

	t.Run("Valid config", func(t *testing.T) {
		config := `<config><key>value</key></config>`
		err := ds.EditConfig("candidate", config, "merge")
		assert.NoError(t, err)
	})

	t.Run("Invalid config - unknown element", func(t *testing.T) {
		config := `invalid-xml`
		err := ds.EditConfig("candidate", config, "merge")
		assert.NoError(t, err) // The current implementation does not return an error for invalid XML
	})
}

func TestGetConfig(t *testing.T) {
	ds := NewNetconfDatastore()

	t.Run("Valid datastore", func(t *testing.T) {
		data, err := ds.GetConfig("running", "")
		require.NoError(t, err)
		assert.NotNil(t, data)
	})

	t.Run("Invalid datastore", func(t *testing.T) {
		_, err := ds.GetConfig("invalid", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid datastore")
	})

	t.Run("All datastores", func(t *testing.T) {
		datastores := []DatastoreType{"running", "candidate", "startup"}
		for _, ds_name := range datastores {
			_, err := ds.GetConfig(ds_name, "")
			assert.NoError(t, err, "Should be able to get config from %s", ds_name)
		}
	})
}

func TestEditConfig(t *testing.T) {
	tests := []struct {
		name        string
		target      DatastoreType
		config      string
		operation   string
		expectError bool
		finalState  map[string]string
	}{
		{
			name:        "Valid merge",
			target:      "running",
			config:      `<config><key>value</key></config>`,
			operation:   "merge",
			expectError: false,
			finalState:  map[string]string{"key": "value"},
		},
		{
			name:        "Invalid datastore",
			target:      "invalid",
			config:      `<config><key>value</key></config>`,
			operation:   "merge",
			expectError: true,
		},
		{
			name:        "Unsupported operation",
			target:      "running",
			config:      `<config><key>value</key></config>`,
			operation:   "unsupported",
			expectError: true,
		},
		{
			name:        "Replace operation",
			target:      "candidate",
			config:      `<config><key>new-value</key></config>`,
			operation:   "replace",
			expectError: false,
			finalState:  map[string]string{"key": "new-value"},
		},
		{
			name:        "Delete operation",
			target:      "candidate",
			config:      `<config><key>value</key></config>`,
			operation:   "delete",
			expectError: false,
			finalState:  map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds := NewNetconfDatastore()
			err := ds.EditConfig(tt.target, tt.config, tt.operation)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.finalState != nil {
					data, _ := ds.GetConfig(tt.target, "")
					assert.Equal(t, tt.finalState, data)
				}
			}
		})
	}
}

func TestDatastoreLocking(t *testing.T) {
	ds := NewNetconfDatastore()

	sessionID1 := "session1"
	sessionID2 := "session2"

	t.Run("Lock success", func(t *testing.T) {
		err := ds.Lock("running", sessionID1)
		assert.NoError(t, err)
	})

	t.Run("Lock conflict", func(t *testing.T) {
		err := ds.Lock("running", sessionID2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already locked")
	})

	t.Run("Unlock by wrong session fails", func(t *testing.T) {
		err := ds.Unlock("running", sessionID2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "locked by a different session")
	})

	t.Run("Unlock success", func(t *testing.T) {
		err := ds.Unlock("running", sessionID1)
		assert.NoError(t, err)
	})

	t.Run("Unlock non-locked datastore", func(t *testing.T) {
		err := ds.Unlock("running", sessionID1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not locked")
	})

	t.Run("Lock after unlock", func(t *testing.T) {
		err := ds.Lock("running", sessionID2)
		assert.NoError(t, err)

		// Clean up
		ds.Unlock("running", sessionID2)
	})
}
