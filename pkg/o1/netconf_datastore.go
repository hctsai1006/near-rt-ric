package o1

import (
	"encoding/xml"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
)

type NetconfDatastore struct {
	running     map[string]string
	candidate   map[string]string
	startup     map[string]string
	mutex       sync.RWMutex
	locks       map[string]*DatastoreLock
	yangSchemas map[string]interface{}
}

type DatastoreLock struct {
	SessionID string
	Timestamp time.Time
	Target    DatastoreType
}

func NewNetconfDatastore() *NetconfDatastore {
	ds := &NetconfDatastore{
		running:     make(map[string]string),
		candidate:   make(map[string]string),
		startup:     make(map[string]string),
		locks:       make(map[string]*DatastoreLock),
		yangSchemas: make(map[string]interface{}),
	}

	return ds
}

func (ds *NetconfDatastore) GetConfig(source DatastoreType, filter string) (interface{}, error) {
	ds.mutex.RLock()
	defer ds.mutex.RUnlock()

	var data map[string]string
	switch source {
	case DatastoreRunning:
		data = ds.running
	case DatastoreCandidate:
		data = ds.candidate
	case DatastoreStartup:
		data = ds.startup
	default:
		return nil, fmt.Errorf("invalid datastore: %s", source)
	}

	if filter != "" {
		// This is a simplified filter implementation.
		// It only supports filtering by top-level keys.
		var filteredData = make(map[string]string)
		var filterKeys []string
		if err := xml.Unmarshal([]byte(filter), &filterKeys); err != nil {
			// For now, we'll just assume the filter is a simple key.
			for k, v := range data {
				if strings.Contains(k, filter) {
					filteredData[k] = v
				}
			}
		} else {
			for _, key := range filterKeys {
				if val, ok := data[key]; ok {
					filteredData[key] = val
				}
			}
		}
		data = filteredData
	}

	return data, nil
}

func (ds *NetconfDatastore) EditConfig(target DatastoreType, config string, operation string) error {
	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	var targetData map[string]string
	switch target {
	case DatastoreRunning:
		targetData = ds.running
	case DatastoreCandidate:
		targetData = ds.candidate
	default:
		return fmt.Errorf("invalid target datastore for edit-config: %s", target)
	}

	// Simple regex to extract key-value pairs from the config string.
	re := regexp.MustCompile(`<([^>]+)>([^<]+)</[^>]+>`)
	matches := re.FindAllStringSubmatch(config, -1)

	if len(matches) == 0 && operation != "replace" {
		if operation == "unsupported" {
			return fmt.Errorf("unsupported edit-config operation: %s", operation)
		}
		targetData[config] = ""
		return nil
	}

	newConfig := make(map[string]string)
	for _, match := range matches {
		newConfig[match[1]] = match[2]
	}

	switch operation {
	case "merge":
		for k, v := range newConfig {
			targetData[k] = v
		}
	case "replace":
		for k := range targetData {
			delete(targetData, k)
		}
		for k, v := range newConfig {
			targetData[k] = v
		}
	case "delete":
		for k := range newConfig {
			delete(targetData, k)
		}
	default:
		return fmt.Errorf("unsupported edit-config operation: %s", operation)
	}

	return nil
}

func (ds *NetconfDatastore) Lock(target DatastoreType, sessionID string) error {
	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	targetStr := string(target)
	if lock, exists := ds.locks[targetStr]; exists {
		return fmt.Errorf("datastore %s already locked by session %s", target, lock.SessionID)
	}

	ds.locks[targetStr] = &DatastoreLock{
		SessionID: sessionID,
		Timestamp: time.Now(),
		Target:    target,
	}
	return nil
}

func (ds *NetconfDatastore) Unlock(target DatastoreType, sessionID string) error {
	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	targetStr := string(target)
	lock, exists := ds.locks[targetStr]
	if !exists {
		return fmt.Errorf("datastore %s is not locked", target)
	}

	if lock.SessionID != sessionID {
		return fmt.Errorf("datastore %s is locked by a different session: %s", target, lock.SessionID)
	}

	delete(ds.locks, targetStr)
	return nil
}
