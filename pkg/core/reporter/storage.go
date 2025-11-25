package reporter

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"sync"

	"github.com/yourusername/infrastructure-resilience-engine/pkg/core/types"
)

// InMemoryStorage is an in-memory implementation of StorageBackend
type InMemoryStorage struct {
	mu          sync.RWMutex
	collections map[string]map[string]interface{}
}

// NewInMemoryStorage creates a new InMemoryStorage instance
func NewInMemoryStorage() *InMemoryStorage {
	return &InMemoryStorage{
		collections: make(map[string]map[string]interface{}),
	}
}

// Save stores a value with the given key
func (s *InMemoryStorage) Save(ctx context.Context, key string, value interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Extract collection from key (format: "collection:id")
	collection, id := parseKey(key)

	if s.collections[collection] == nil {
		s.collections[collection] = make(map[string]interface{})
	}

	s.collections[collection][id] = value
	return nil
}

// Load retrieves a value by key
func (s *InMemoryStorage) Load(ctx context.Context, key string) (interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	collection, id := parseKey(key)

	if s.collections[collection] == nil {
		return nil, fmt.Errorf("collection %s not found", collection)
	}

	value, ok := s.collections[collection][id]
	if !ok {
		return nil, fmt.Errorf("key %s not found", key)
	}

	return value, nil
}

// Query retrieves values matching the query criteria
func (s *InMemoryStorage) Query(ctx context.Context, query types.Query) ([]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	collection := s.collections[query.Collection]
	if collection == nil {
		return []interface{}{}, nil
	}

	// Collect all values
	var results []interface{}
	for _, value := range collection {
		if matchesFilter(value, query.Filter) {
			results = append(results, value)
		}
	}

	// Sort results
	if len(query.Sort) > 0 {
		sortResults(results, query.Sort)
	}

	// Apply pagination
	start := query.Offset
	if start > len(results) {
		start = len(results)
	}

	end := start + query.Limit
	if query.Limit == 0 || end > len(results) {
		end = len(results)
	}

	return results[start:end], nil
}

// Delete removes a value by key
func (s *InMemoryStorage) Delete(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	collection, id := parseKey(key)

	if s.collections[collection] == nil {
		return fmt.Errorf("collection %s not found", collection)
	}

	delete(s.collections[collection], id)
	return nil
}

// Close closes the storage (no-op for in-memory)
func (s *InMemoryStorage) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.collections = make(map[string]map[string]interface{})
	return nil
}

// parseKey splits a key into collection and id
func parseKey(key string) (collection, id string) {
	// Simple parsing: "collection:id"
	for i, c := range key {
		if c == ':' {
			return key[:i], key[i+1:]
		}
	}
	// If no colon, use "default" as collection
	return "default", key
}

// matchesFilter checks if a value matches the filter criteria
func matchesFilter(value interface{}, filter map[string]interface{}) bool {
	if len(filter) == 0 {
		return true
	}

	v := reflect.ValueOf(value)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return false
	}

	for key, filterValue := range filter {
		field := v.FieldByName(key)
		if !field.IsValid() {
			return false
		}

		// Compare values
		if !reflect.DeepEqual(field.Interface(), filterValue) {
			return false
		}
	}

	return true
}

// sortResults sorts the results based on sort fields
func sortResults(results []interface{}, sortFields []types.SortField) {
	if len(results) == 0 || len(sortFields) == 0 {
		return
	}

	sort.Slice(results, func(i, j int) bool {
		vi := reflect.ValueOf(results[i])
		vj := reflect.ValueOf(results[j])

		if vi.Kind() == reflect.Ptr {
			vi = vi.Elem()
		}
		if vj.Kind() == reflect.Ptr {
			vj = vj.Elem()
		}

		for _, sf := range sortFields {
			fi := vi.FieldByName(sf.Field)
			fj := vj.FieldByName(sf.Field)

			if !fi.IsValid() || !fj.IsValid() {
				continue
			}

			cmp := compareValues(fi.Interface(), fj.Interface())
			if cmp != 0 {
				if sf.Descending {
					return cmp > 0
				}
				return cmp < 0
			}
		}

		return false
	})
}

// compareValues compares two values and returns -1, 0, or 1
func compareValues(a, b interface{}) int {
	switch va := a.(type) {
	case int:
		vb, ok := b.(int)
		if !ok {
			return 0
		}
		if va < vb {
			return -1
		} else if va > vb {
			return 1
		}
		return 0
	case int64:
		vb, ok := b.(int64)
		if !ok {
			return 0
		}
		if va < vb {
			return -1
		} else if va > vb {
			return 1
		}
		return 0
	case float64:
		vb, ok := b.(float64)
		if !ok {
			return 0
		}
		if va < vb {
			return -1
		} else if va > vb {
			return 1
		}
		return 0
	case string:
		vb, ok := b.(string)
		if !ok {
			return 0
		}
		if va < vb {
			return -1
		} else if va > vb {
			return 1
		}
		return 0
	default:
		return 0
	}
}
