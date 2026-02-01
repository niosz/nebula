package storage

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	bolt "go.etcd.io/bbolt"
)

// Storage manages the BoltDB database
type Storage struct {
	db *bolt.DB
	mu sync.RWMutex
}

// New creates a new Storage instance
func New(path string) (*Storage, error) {
	db, err := bolt.Open(path, 0600, &bolt.Options{
		Timeout: 5 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	s := &Storage{db: db}

	// Initialize all buckets
	if err := s.initBuckets(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize buckets: %w", err)
	}

	return s, nil
}

// Close closes the database
func (s *Storage) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.db.Close()
}

// DB returns the underlying bolt.DB instance
func (s *Storage) DB() *bolt.DB {
	return s.db
}

// Get retrieves a value from a bucket
func (s *Storage) Get(bucket, key string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var value []byte
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucket)
		}
		v := b.Get([]byte(key))
		if v != nil {
			value = make([]byte, len(v))
			copy(value, v)
		}
		return nil
	})
	return value, err
}

// Set stores a value in a bucket
func (s *Storage) Set(bucket, key string, value []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucket)
		}
		return b.Put([]byte(key), value)
	})
}

// Delete removes a value from a bucket
func (s *Storage) Delete(bucket, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucket)
		}
		return b.Delete([]byte(key))
	})
}

// GetJSON retrieves and unmarshals a JSON value
func (s *Storage) GetJSON(bucket, key string, v interface{}) error {
	data, err := s.Get(bucket, key)
	if err != nil {
		return err
	}
	if data == nil {
		return nil
	}
	return json.Unmarshal(data, v)
}

// SetJSON marshals and stores a JSON value
func (s *Storage) SetJSON(bucket, key string, v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}
	return s.Set(bucket, key, data)
}

// GetAll retrieves all key-value pairs from a bucket
func (s *Storage) GetAll(bucket string) (map[string][]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string][]byte)
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucket)
		}
		return b.ForEach(func(k, v []byte) error {
			value := make([]byte, len(v))
			copy(value, v)
			result[string(k)] = value
			return nil
		})
	})
	return result, err
}

// DeleteOlderThan removes entries older than the specified duration
// Requires entries to have a "timestamp" field in JSON format
func (s *Storage) DeleteOlderThan(bucket string, duration time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-duration)
	var keysToDelete [][]byte

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucket)
		}
		return b.ForEach(func(k, v []byte) error {
			var entry struct {
				Timestamp time.Time `json:"timestamp"`
			}
			if err := json.Unmarshal(v, &entry); err == nil {
				if entry.Timestamp.Before(cutoff) {
					key := make([]byte, len(k))
					copy(key, k)
					keysToDelete = append(keysToDelete, key)
				}
			}
			return nil
		})
	})
	if err != nil {
		return err
	}

	if len(keysToDelete) == 0 {
		return nil
	}

	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		for _, key := range keysToDelete {
			if err := b.Delete(key); err != nil {
				return err
			}
		}
		return nil
	})
}

// Count returns the number of entries in a bucket
func (s *Storage) Count(bucket string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var count int
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucket)
		}
		count = b.Stats().KeyN
		return nil
	})
	return count, err
}
