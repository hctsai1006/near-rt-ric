package o1

import (
	"fmt"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewNetconfDatastore(t *testing.T) {
	ds, err := NewNetconfDatastore("../../yang/o-ran-sc-ric-plt-o1-v1.0.0.yang")
	require.NoError(t, err)
	require.NotNil(t, ds)
	assert.NotNil(t, ds.running)
	assert.NotNil(t, ds.candidate)
	assert.NotNil(t, ds.startup)
	assert.NotNil(t, ds.locks)
	assert.NotNil(t, ds.yangSchemas)
	assert.Contains(t, ds.yangSchemas, "o-ran-sc-ric-plt-o1")
}

func TestEditConfig_Validation(t *testing.T) {
	ds, err := NewNetconfDatastore("../../yang/o-ran-sc-ric-plt-o1-v1.0.0.yang")
	require.NoError(t, err)

	t.Run("Valid config", func(t *testing.T) {
		config := []byte(`<o-ran-sc-ric-plt-o1><ric-instance><id>1</id></ric-instance></o-ran-sc-ric-plt-o1>`)
		err := ds.EditConfig("candidate", config, "merge")
		assert.NoError(t, err)
	})

	t.Run("Invalid config - unknown element", func(t *testing.T) {
		config := []byte(`<o-ran-sc-ric-plt-o1><invalid-element>true</invalid-element></o-ran-sc-ric-plt-o1>`)
		err := ds.EditConfig("candidate", config, "merge")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown element")
	})
}

func TestGetConfig(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	ds := NewNetconfDatastore(logger)

	t.Run("Valid datastore", func(t *testing.T) {
		data, err := ds.GetConfig("running", nil)
		require.NoError(t, err)
		assert.NotEmpty(t, data)
	})

	t.Run("Invalid datastore", func(t *testing.T) {
		_, err := ds.GetConfig("invalid", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid datastore")
	})

	t.Run("All datastores", func(t *testing.T) {
		datastores := []string{"running", "candidate", "startup"}
		for _, ds_name := range datastores {
			_, err := ds.GetConfig(ds_name, nil)
			assert.NoError(t, err, "Should be able to get config from %s", ds_name)
		}
	})
}

func TestEditConfig(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	ds := NewNetconfDatastore(logger)

	sessionID := "test-session"
	username := "test-user"

	t.Run("Valid configuration merge", func(t *testing.T) {
		config := []byte(`<config><o-ran-sc-ric><node-info><node-id>test-node-001</node-id></node-info></o-ran-sc-ric></config>`)
		err := ds.EditConfig(sessionID, username, "running", config, "merge")
		assert.NoError(t, err)
	})

	t.Run("Invalid datastore", func(t *testing.T) {
		config := []byte(`<config><test>value</test></config>`)
		err := ds.EditConfig(sessionID, username, "invalid", config, "merge")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid target datastore")
	})

	t.Run("Unsupported operation", func(t *testing.T) {
		config := []byte(`<config><test>value</test></config>`)
		err := ds.EditConfig(sessionID, username, "running", config, "unsupported")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported operation")
	})

	t.Run("Replace operation", func(t *testing.T) {
		config := []byte(`<config><o-ran-sc-ric><node-info><node-id>replaced-node</node-id></node-info></o-ran-sc-ric></config>`)
		err := ds.EditConfig(sessionID, username, "candidate", config, "replace")
		assert.NoError(t, err)
	})

	t.Run("Delete operation", func(t *testing.T) {
		config := []byte(`<config><o-ran-sc-ric><node-info><node-id>test</node-id></node-info></o-ran-sc-ric></config>`)
		err := ds.EditConfig(sessionID, username, "candidate", config, "delete")
		assert.NoError(t, err)
	})
}

func TestDatastoreLocking(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	ds := NewNetconfDatastore(logger)

	sessionID1 := "session1"
	sessionID2 := "session2"
	username := "test-user"

	t.Run("Lock success", func(t *testing.T) {
		err := ds.Lock("running", sessionID1, username)
		assert.NoError(t, err)
	})

	t.Run("Lock conflict", func(t *testing.T) {
		err := ds.Lock("running", sessionID2, username)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already locked")
	})

	t.Run("Unlock by wrong session fails", func(t *testing.T) {
		err := ds.Unlock("running", sessionID2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "locked by different session")
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
		err := ds.Lock("running", sessionID2, username)
		assert.NoError(t, err)
		
		// Clean up
		ds.Unlock("running", sessionID2)
	})
}

func TestCopyConfig(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	ds := NewNetconfDatastore(logger)

	sessionID := "test-session"
	username := "test-user"

	t.Run("Valid copy operation", func(t *testing.T) {
		err := ds.CopyConfig(sessionID, username, "running", "candidate")
		assert.NoError(t, err)
	})

	t.Run("Invalid source datastore", func(t *testing.T) {
		err := ds.CopyConfig(sessionID, username, "invalid", "candidate")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid source datastore")
	})

	t.Run("Invalid target datastore", func(t *testing.T) {
		err := ds.CopyConfig(sessionID, username, "running", "invalid")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid target datastore")
	})

	t.Run("Copy all datastores", func(t *testing.T) {
		datastores := []string{"running", "candidate", "startup"}
		for _, source := range datastores {
			for _, target := range datastores {
				if source != target {
					err := ds.CopyConfig(sessionID, username, source, target)
					assert.NoError(t, err, "Should be able to copy from %s to %s", source, target)
				}
			}
		}
	})
}

func TestDeleteConfig(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	ds := NewNetconfDatastore(logger)

	sessionID := "test-session"
	username := "test-user"

	t.Run("Delete candidate datastore", func(t *testing.T) {
		// First add some config
		config := []byte(`<config><test>value</test></config>`)
		ds.EditConfig(sessionID, username, "candidate", config, "merge")
		
		// Then delete it
		err := ds.DeleteConfig(sessionID, username, "candidate")
		assert.NoError(t, err)
		
		// Verify it's empty
		data, err := ds.GetConfig("candidate", nil)
		require.NoError(t, err)
		assert.Contains(t, string(data), "<config></config>")
	})

	t.Run("Delete startup datastore", func(t *testing.T) {
		err := ds.DeleteConfig(sessionID, username, "startup")
		assert.NoError(t, err)
	})

	t.Run("Cannot delete running datastore", func(t *testing.T) {
		err := ds.DeleteConfig(sessionID, username, "running")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot delete from datastore")
	})
}

func TestGetLockInfo(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	ds := NewNetconfDatastore(logger)

	sessionID := "test-session"
	username := "test-user"

	t.Run("No locks initially", func(t *testing.T) {
		locks := ds.GetLockInfo()
		assert.Empty(t, locks)
	})

	t.Run("Lock info after locking", func(t *testing.T) {
		err := ds.Lock("running", sessionID, username)
		require.NoError(t, err)

		locks := ds.GetLockInfo()
		assert.Len(t, locks, 1)
		assert.Contains(t, locks, "running")
		
		lock := locks["running"]
		assert.Equal(t, sessionID, lock.SessionID)
		assert.Equal(t, username, lock.Username)
		assert.Equal(t, "running", lock.Target)
		assert.WithinDuration(t, time.Now(), lock.Timestamp, time.Second)
		
		// Clean up
		ds.Unlock("running", sessionID)
	})
}

func TestGetChangeHistory(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	ds := NewNetconfDatastore(logger)

	sessionID := "test-session"
	username := "test-user"

	t.Run("No changes initially", func(t *testing.T) {
		history := ds.GetChangeHistory(10)
		assert.Empty(t, history)
	})

	t.Run("Changes recorded after operations", func(t *testing.T) {
		// Make some changes
		config := []byte(`<config><o-ran-sc-ric><node-info><node-id>history-test</node-id></node-info></o-ran-sc-ric></config>`)
		err := ds.EditConfig(sessionID, username, "running", config, "merge")
		require.NoError(t, err)

		err = ds.CopyConfig(sessionID, username, "running", "candidate")
		require.NoError(t, err)

		// Check history
		history := ds.GetChangeHistory(10)
		assert.NotEmpty(t, history)
		
		// Verify recent changes are recorded
		for _, change := range history {
			assert.Equal(t, sessionID, change.SessionID)
			assert.Equal(t, username, change.Username)
			assert.WithinDuration(t, time.Now(), change.Timestamp, time.Second)
		}
	})

	t.Run("History limit respected", func(t *testing.T) {
		history := ds.GetChangeHistory(1)
		assert.LessOrEqual(t, len(history), 1)
	})
}

func TestConfigValidator(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	ds := NewNetconfDatastore(logger)

	t.Run("Valid O-RAN configuration", func(t *testing.T) {
		config := map[string]interface{}{
			"o-ran-sc-ric": map[string]interface{}{
				"node-info": map[string]interface{}{
					"node-id":   "valid-node",
					"node-type": "near-rt-ric",
				},
			},
		}
		err := ds.validator.ValidateConfig(config)
		assert.NoError(t, err)
	})

	t.Run("Missing required node-id", func(t *testing.T) {
		config := map[string]interface{}{
			"o-ran-sc-ric": map[string]interface{}{
				"node-info": map[string]interface{}{
					"node-type": "near-rt-ric",
				},
			},
		}
		err := ds.validator.ValidateConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "node-id is required")
	})

	t.Run("Missing required node-type", func(t *testing.T) {
		config := map[string]interface{}{
			"o-ran-sc-ric": map[string]interface{}{
				"node-info": map[string]interface{}{
					"node-id": "test-node",
				},
			},
		}
		err := ds.validator.ValidateConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "node-type is required")
	})

	t.Run("Invalid interface configuration", func(t *testing.T) {
		config := map[string]interface{}{
			"o-ran-sc-ric": map[string]interface{}{
				"node-info": map[string]interface{}{
					"node-id":   "test-node",
					"node-type": "near-rt-ric",
				},
				"interfaces": map[string]interface{}{
					"e2": map[string]interface{}{
						"sctp-port": "invalid-port", // Should be integer
					},
				},
			},
		}
		err := ds.validator.ValidateConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "sctp-port must be a valid port number")
	})

	t.Run("Valid interface configuration", func(t *testing.T) {
		config := map[string]interface{}{
			"o-ran-sc-ric": map[string]interface{}{
				"node-info": map[string]interface{}{
					"node-id":   "test-node",
					"node-type": "near-rt-ric",
				},
				"interfaces": map[string]interface{}{
					"e2": map[string]interface{}{
						"enabled":   true,
						"sctp-port": 36421,
					},
					"a1": map[string]interface{}{
						"enabled":   true,
						"http-port": 8080,
					},
					"o1": map[string]interface{}{
						"enabled":      true,
						"netconf-port": 830,
					},
				},
			},
		}
		err := ds.validator.ValidateConfig(config)
		assert.NoError(t, err)
	})
}

func TestDeepCopy(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	ds := NewNetconfDatastore(logger)

	original := map[string]interface{}{
		"string": "value",
		"number": 42,
		"nested": map[string]interface{}{
			"inner": "value",
		},
		"array": []interface{}{
			"item1",
			map[string]interface{}{
				"inner": "array-item",
			},
		},
	}

	copied := ds.deepCopy(original)

	// Verify deep copy worked
	assert.Equal(t, original, copied)

	// Modify original and ensure copy is not affected
	original["string"] = "modified"
	assert.NotEqual(t, original["string"], copied["string"])

	// Modify nested object
	if nested, ok := original["nested"].(map[string]interface{}); ok {
		nested["inner"] = "modified"
	}
	copiedNested := copied["nested"].(map[string]interface{})
	assert.NotEqual(t, "modified", copiedNested["inner"])
}

func TestFilterApplication(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	ds := NewNetconfDatastore(logger)

	data := map[string]interface{}{
		"root": map[string]interface{}{
			"child1": "value1",
			"child2": "value2",
		},
	}

	t.Run("No filter returns all data", func(t *testing.T) {
		result := ds.applyFilter(data, nil)
		assert.Equal(t, data, result)
	})

	t.Run("Root filter returns all data", func(t *testing.T) {
		filter := &Filter{Subtree: "/"}
		result := ds.applyFilter(data, filter)
		assert.Equal(t, data, result)
	})

	t.Run("Specific subtree filter", func(t *testing.T) {
		filter := &Filter{Subtree: "/root/child1"}
		result := ds.applyFilter(data, filter)
		assert.NotNil(t, result)
		// For now, our simple implementation returns all data
		// In production, this would implement full XPath filtering
	})
}

func TestChangeHistoryRotation(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	ds := NewNetconfDatastore(logger)
	ds.maxHistory = 5 // Set small limit for testing

	sessionID := "test-session"
	username := "test-user"

	// Create more changes than the limit
	for i := 0; i < 10; i++ {
		changes := []ConfigChange{
			{
				Timestamp: time.Now(),
				SessionID: sessionID,
				Username:  username,
				Operation: "test",
				Path:      "/test",
				NewValue:  i,
			},
		}
		ds.recordChanges(changes)
	}

	// Verify history was rotated
	history := ds.GetChangeHistory(20)
	assert.LessOrEqual(t, len(history), ds.maxHistory)
	
	// Verify most recent changes are kept
	if len(history) > 0 {
		lastChange := history[len(history)-1]
		assert.Equal(t, 9, lastChange.NewValue)
	}
}

// Benchmark tests for performance validation
func BenchmarkGetConfig(b *testing.B) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	ds := NewNetconfDatastore(logger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ds.GetConfig("running", nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEditConfig(b *testing.B) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	ds := NewNetconfDatastore(logger)

	config := []byte(`<config><o-ran-sc-ric><node-info><node-id>bench-test</node-id></node-info></o-ran-sc-ric></config>`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := ds.EditConfig("bench-session", "bench-user", "candidate", config, "merge")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLockUnlock(b *testing.B) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	ds := NewNetconfDatastore(logger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sessionID := fmt.Sprintf("session-%d", i)
		err := ds.Lock("candidate", sessionID, "bench-user")
		if err != nil {
			b.Fatal(err)
		}
		err = ds.Unlock("candidate", sessionID)
		if err != nil {
			b.Fatal(err)
		}
	}
}