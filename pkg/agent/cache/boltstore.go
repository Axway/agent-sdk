package cache

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	pkgcache "github.com/Axway/agent-sdk/pkg/cache"
	"github.com/Axway/agent-sdk/pkg/util"
	"go.etcd.io/bbolt"
)

// Error definitions for cache operations
var (
	// ErrReadOnlyMode is returned when a write operation is attempted on a read-only database
	ErrReadOnlyMode = errors.New("cannot perform write operation: database is in read-only mode")
	// ErrDatabaseNotAvailable is returned when the database is not initialized
	ErrDatabaseNotAvailable = errors.New("database not available")
	// ErrBucketNotFound is returned when a required bucket does not exist
	ErrBucketNotFound = errors.New("bucket not found")
)

// boltStore error handling guide:
//
// Write operations (Set, SetWithSecondaryKey, SetWithForeignKey, SetSecondaryKey, SetForeignKey,
// Delete, DeleteBySecondaryKey, DeleteSecondaryKey, DeleteItemsByForeignKey, Flush) may return:
//   - ErrReadOnlyMode: The database is in read-only mode. This occurs when the database file
//     is on a read-only filesystem or the cache was opened with ReadOnly flag. In this state,
//     only read operations (Get, GetBySecondaryKey, etc.) are allowed.
//   - ErrDatabaseNotAvailable: The database handle is nil. The cache was initialized without
//     a database connection.
//   - Other errors from bbolt or serialization failures (JSON/gob encoding errors).
//
// Read operations (Get, GetBySecondaryKey, GetItemsByForeignKey, etc.) may return:
//   - Key not found errors: The requested key/secondary key/foreign key does not exist.
//   - Serialization errors: Data in the database is corrupted or cannot be deserialized.
//   - ErrDatabaseNotAvailable: The database handle is nil.
//
// Flush() operates gracefully in read-only mode by becoming a no-op and logging to stderr.

// recordEnvelope wraps stored data with metadata
type recordEnvelope struct {
	Data               []byte                 // Serialized data
	Hash               uint64                 // Hash for change detection
	UpdateTime         int64                  // Unix timestamp
	Aliases            map[string]bool        // All secondary keys pointing to this item
	GroupKey           string                 // Foreign key for grouping
	IsGobEncoded       bool                   // True if data is gob-encoded, false if JSON
	IsResourceInstance bool                   `json:"isResourceInstance"`
	HasRawResource     bool                   `json:"hasRawResource"`
	RawResource        json.RawMessage        `json:"rawResource,omitempty"`
	Owner              *v1.Owner              `json:"owner,omitempty"`
	SubResources       map[string]interface{} `json:"subResources,omitempty"`
	SubResourceHashes  map[string]interface{} `json:"subResourceHashes,omitempty"`
}

// boltStore implements storage using bbolt for persistence
type boltStore struct {
	db         *bbolt.DB
	bucketName string
}

// ensureWritable checks if the database is available and not in read-only mode
// Returns an error if writes cannot be performed
func (bc *boltStore) ensureWritable() error {
	if bc.db == nil {
		return ErrDatabaseNotAvailable
	}
	if bc.db.IsReadOnly() {
		return ErrReadOnlyMode
	}
	return nil
}

// newBoltStore creates a new bbolt-backed storage instance for the given bucket
func newBoltStore(db *bbolt.DB, bucketName string) (*boltStore, error) {
	bs := &boltStore{
		db:         db,
		bucketName: bucketName,
	}

	// Initialize buckets
	err := db.Update(func(tx *bbolt.Tx) error {
		// Create primary bucket
		if _, err := tx.CreateBucketIfNotExists([]byte(bucketName)); err != nil {
			return fmt.Errorf("failed to create bucket %s: %w", bucketName, err)
		}
		// Create secondary key index bucket
		if _, err := tx.CreateBucketIfNotExists([]byte(bucketName + "_secondary")); err != nil {
			return fmt.Errorf("failed to create secondary index bucket: %w", err)
		}
		// Create foreign key index bucket
		if _, err := tx.CreateBucketIfNotExists([]byte(bucketName + "_foreign")); err != nil {
			return fmt.Errorf("failed to create foreign index bucket: %w", err)
		}
		return nil
	})

	return bs, err
}

// serialize encodes data into bytes, attempting gob first, then falling back to JSON
func serialize(data interface{}) ([]byte, bool, error) {
	// Try gob encoding first (more efficient for complex types)
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(data); err == nil {
		return buf.Bytes(), true, nil
	}

	// Fall back to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, false, fmt.Errorf("failed to serialize data: %w", err)
	}
	return jsonData, false, nil
}

// decodeRecord decodes a record envelope into the original object, preserving ResourceInstance metadata
func decodeRecord(envelope recordEnvelope) (interface{}, error) {
	if envelope.IsResourceInstance || envelope.HasRawResource || envelope.SubResources != nil || envelope.SubResourceHashes != nil {
		var ri v1.ResourceInstance
		if envelope.HasRawResource && len(envelope.RawResource) > 0 {
			if err := ri.UnmarshalJSON(envelope.RawResource); err != nil {
				return nil, err
			}
		} else {
			type resourceInstanceNoRaw v1.ResourceInstance
			var riNoRaw resourceInstanceNoRaw
			if err := json.Unmarshal(envelope.Data, &riNoRaw); err != nil {
				return nil, fmt.Errorf("failed to json decode: %w", err)
			}
			ri = v1.ResourceInstance(riNoRaw)
			if !envelope.HasRawResource {
				ri.Metadata.Audit.CreateTimestamp = v1.Time{}
				ri.Metadata.Audit.ModifyTimestamp = v1.Time{}
			}
		}
		if ri.Owner == nil && envelope.Owner != nil {
			ri.Owner = envelope.Owner
		}
		if envelope.SubResources != nil {
			ri.SubResources = envelope.SubResources
		}
		if envelope.SubResourceHashes != nil {
			ri.SubResourceHashes = envelope.SubResourceHashes
		}
		return &ri, nil
	}

	if envelope.IsGobEncoded {
		var result interface{}
		buf := bytes.NewBuffer(envelope.Data)
		dec := gob.NewDecoder(buf)
		if err := dec.Decode(&result); err == nil {
			return result, nil
		}

		buf = bytes.NewBuffer(envelope.Data)
		dec = gob.NewDecoder(buf)
		var seqID int64
		if err := dec.Decode(&seqID); err == nil {
			return seqID, nil
		}

		buf = bytes.NewBuffer(envelope.Data)
		dec = gob.NewDecoder(buf)
		var seqIDInt int
		if err := dec.Decode(&seqIDInt); err == nil {
			return seqIDInt, nil
		}

		buf = bytes.NewBuffer(envelope.Data)
		dec = gob.NewDecoder(buf)
		var svcCount apiServiceToInstanceCount
		if err := dec.Decode(&svcCount); err == nil {
			return svcCount, nil
		}

		buf = bytes.NewBuffer(envelope.Data)
		dec = gob.NewDecoder(buf)
		var team defs.PlatformTeam
		if err := dec.Decode(&team); err == nil {
			return team, nil
		}

		return nil, fmt.Errorf("failed to gob decode")
	}

	var result interface{}
	if err := json.Unmarshal(envelope.Data, &result); err != nil {
		return nil, fmt.Errorf("failed to json decode: %w", err)
	}
	return result, nil
}

func addResourceInstanceFields(envelope *recordEnvelope, data interface{}) {
	if envelope == nil || data == nil {
		return
	}

	if ri, ok := data.(*v1.ResourceInstance); ok && ri != nil {
		envelope.IsResourceInstance = true
		rawResource := ri.GetRawResource()
		envelope.RawResource = rawResource
		envelope.HasRawResource = rawResource != nil
		envelope.Owner = ri.Owner
		envelope.SubResources = ri.SubResources
		envelope.SubResourceHashes = ri.SubResourceHashes
		return
	}

	if ri, ok := data.(v1.ResourceInstance); ok {
		envelope.IsResourceInstance = true
		rawResource := ri.GetRawResource()
		envelope.RawResource = rawResource
		envelope.HasRawResource = rawResource != nil
		envelope.Owner = ri.Owner
		envelope.SubResources = ri.SubResources
		envelope.SubResourceHashes = ri.SubResourceHashes
	}
}

// Get retrieves data by primary key
func (bc *boltStore) Get(key string) (interface{}, error) {
	if bc.db == nil {
		return nil, fmt.Errorf("database not available")
	}
	var result interface{}
	err := bc.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(bc.bucketName))
		if bucket == nil {
			return fmt.Errorf("bucket not found")
		}

		data := bucket.Get([]byte(key))
		if data == nil {
			return fmt.Errorf("key not found: %s", key)
		}

		// Deserialize envelope
		var envelope recordEnvelope
		if err := json.Unmarshal(data, &envelope); err != nil {
			return fmt.Errorf("failed to unmarshal envelope: %w", err)
		}

		// Deserialize data
		var err error
		result, err = decodeRecord(envelope)
		return err
	})

	return result, err
}

// Set stores data with primary key
func (bc *boltStore) Set(key string, data interface{}) error {
	return bc.SetWithSecondaryKey(key, "", data)
}

// SetWithSecondaryKey stores data with primary and secondary key
func (bc *boltStore) SetWithSecondaryKey(key string, secondaryKey string, data interface{}) error {
	if err := bc.ensureWritable(); err != nil {
		return err
	}
	if key == "" {
		if secondaryKey == "" {
			return fmt.Errorf("key not provided")
		}
		key = secondaryKey
	}
	return bc.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(bc.bucketName))
		if bucket == nil {
			return fmt.Errorf("bucket not found")
		}

		// Serialize the data
		serialized, useGob, err := serialize(data)
		if err != nil {
			return err
		}

		// Compute hash for change detection
		hash, _ := util.ComputeHash(data)

		// Get existing envelope to preserve secondary keys
		existingSecondaryKeys := make(map[string]bool)
		if existingData := bucket.Get([]byte(key)); existingData != nil {
			var existingEnvelope recordEnvelope
			if err := json.Unmarshal(existingData, &existingEnvelope); err == nil {
				existingSecondaryKeys = existingEnvelope.Aliases
			}
		}

		// Add new secondary key if provided
		if secondaryKey != "" {
			existingSecondaryKeys[secondaryKey] = true
		}

		// Create envelope
		envelope := recordEnvelope{
			Data:         serialized,
			Hash:         hash,
			UpdateTime:   time.Now().Unix(),
			Aliases:      existingSecondaryKeys,
			IsGobEncoded: useGob,
		}
		addResourceInstanceFields(&envelope, data)

		// Serialize envelope
		envelopeData, err := json.Marshal(envelope)
		if err != nil {
			return fmt.Errorf("failed to marshal envelope: %w", err)
		}

		// Store in primary bucket
		if err := bucket.Put([]byte(key), envelopeData); err != nil {
			return err
		}

		// Store in secondary index if provided
		if secondaryKey != "" {
			secondaryBucket := tx.Bucket([]byte(bc.bucketName + "_secondary"))
			if secondaryBucket == nil {
				return fmt.Errorf("secondary index bucket not found")
			}
			if err := secondaryBucket.Put([]byte(secondaryKey), []byte(key)); err != nil {
				return err
			}
		}

		return nil
	})
}

// SetSecondaryKey adds a secondary key to an existing item
func (bc *boltStore) SetSecondaryKey(key string, secondaryKey string) error {
	if err := bc.ensureWritable(); err != nil {
		return err
	}
	return bc.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(bc.bucketName))
		if bucket == nil {
			return fmt.Errorf("bucket not found")
		}

		// Get existing envelope
		existingData := bucket.Get([]byte(key))
		if existingData == nil {
			return fmt.Errorf("key not found: %s", key)
		}

		var envelope recordEnvelope
		if err := json.Unmarshal(existingData, &envelope); err != nil {
			return fmt.Errorf("failed to unmarshal envelope: %w", err)
		}

		// Add secondary key
		if envelope.Aliases == nil {
			envelope.Aliases = make(map[string]bool)
		}
		envelope.Aliases[secondaryKey] = true

		// Save updated envelope
		envelopeData, err := json.Marshal(envelope)
		if err != nil {
			return err
		}
		if err := bucket.Put([]byte(key), envelopeData); err != nil {
			return err
		}

		// Update secondary index
		secondaryBucket := tx.Bucket([]byte(bc.bucketName + "_secondary"))
		if secondaryBucket == nil {
			return fmt.Errorf("secondary index bucket not found")
		}
		return secondaryBucket.Put([]byte(secondaryKey), []byte(key))
	})
}

// SetWithForeignKey stores data with primary key and foreign key
func (bc *boltStore) SetWithForeignKey(key string, foreignKey string, data interface{}) error {
	if err := bc.ensureWritable(); err != nil {
		return err
	}
	return bc.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(bc.bucketName))
		if bucket == nil {
			return fmt.Errorf("bucket not found")
		}

		// Serialize the data
		serialized, useGob, err := serialize(data)
		if err != nil {
			return err
		}

		// Compute hash
		hash, _ := util.ComputeHash(data)

		// Create envelope with foreign key
		envelope := recordEnvelope{
			Data:         serialized,
			Hash:         hash,
			UpdateTime:   time.Now().Unix(),
			GroupKey:     foreignKey,
			IsGobEncoded: useGob,
		}
		addResourceInstanceFields(&envelope, data)

		// Serialize envelope
		envelopeData, err := json.Marshal(envelope)
		if err != nil {
			return err
		}

		// Store in primary bucket
		if err := bucket.Put([]byte(key), envelopeData); err != nil {
			return err
		}

		// Update foreign key index
		if foreignKey != "" {
			foreignBucket := tx.Bucket([]byte(bc.bucketName + "_foreign"))
			if foreignBucket == nil {
				return fmt.Errorf("foreign index bucket not found")
			}

			// Get existing keys for this foreign key
			var keys []string
			if existingData := foreignBucket.Get([]byte(foreignKey)); existingData != nil {
				if err := json.Unmarshal(existingData, &keys); err != nil {
					keys = []string{}
				}
			}

			// Add this key if not already present
			found := false
			for _, k := range keys {
				if k == key {
					found = true
					break
				}
			}
			if !found {
				keys = append(keys, key)
			}

			// Save updated keys list
			keysData, err := json.Marshal(keys)
			if err != nil {
				return err
			}
			if err := foreignBucket.Put([]byte(foreignKey), keysData); err != nil {
				return err
			}
		}

		return nil
	})
}

// SetForeignKey sets the foreign key for an existing item
func (bc *boltStore) SetForeignKey(key string, foreignKey string) error {
	if err := bc.ensureWritable(); err != nil {
		return err
	}
	return bc.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(bc.bucketName))
		if bucket == nil {
			return fmt.Errorf("bucket not found")
		}

		// Get existing envelope
		existingData := bucket.Get([]byte(key))
		if existingData == nil {
			return fmt.Errorf("key not found: %s", key)
		}

		var envelope recordEnvelope
		if err := json.Unmarshal(existingData, &envelope); err != nil {
			return err
		}

		// Update foreign key
		envelope.GroupKey = foreignKey

		// Save updated envelope
		envelopeData, err := json.Marshal(envelope)
		if err != nil {
			return err
		}
		if err := bucket.Put([]byte(key), envelopeData); err != nil {
			return err
		}

		// Update foreign key index
		foreignBucket := tx.Bucket([]byte(bc.bucketName + "_foreign"))
		if foreignBucket == nil {
			return fmt.Errorf("foreign index bucket not found")
		}

		var keys []string
		if existingData := foreignBucket.Get([]byte(foreignKey)); existingData != nil {
			if err := json.Unmarshal(existingData, &keys); err != nil {
				keys = []string{}
			}
		}

		// Add key if not present
		found := false
		for _, k := range keys {
			if k == key {
				found = true
				break
			}
		}
		if !found {
			keys = append(keys, key)
		}

		keysData, err := json.Marshal(keys)
		if err != nil {
			return err
		}
		return foreignBucket.Put([]byte(foreignKey), keysData)
	})
}

// GetBySecondaryKey retrieves data by secondary key
func (bc *boltStore) GetBySecondaryKey(secondaryKey string) (interface{}, error) {
	var result interface{}
	if bc.db == nil {
		return nil, fmt.Errorf("database not available")
	}
	err := bc.db.View(func(tx *bbolt.Tx) error {
		// Lookup primary key
		secondaryBucket := tx.Bucket([]byte(bc.bucketName + "_secondary"))
		if secondaryBucket == nil {
			return fmt.Errorf("secondary index bucket not found")
		}

		primaryKey := secondaryBucket.Get([]byte(secondaryKey))
		if primaryKey == nil {
			return fmt.Errorf("secondary key not found: %s", secondaryKey)
		}

		// Get data from primary bucket
		bucket := tx.Bucket([]byte(bc.bucketName))
		if bucket == nil {
			return fmt.Errorf("bucket not found")
		}

		data := bucket.Get(primaryKey)
		if data == nil {
			return fmt.Errorf("primary key not found: %s", string(primaryKey))
		}

		// Deserialize envelope
		var envelope recordEnvelope
		if err := json.Unmarshal(data, &envelope); err != nil {
			return err
		}

		// Deserialize data
		var err error
		result, err = decodeRecord(envelope)
		return err
	})

	return result, err
}

// GetItemsByForeignKey retrieves all items with the given foreign key
func (bc *boltStore) GetItemsByForeignKey(foreignKey string) ([]*pkgcache.Item, error) {
	var results []*pkgcache.Item
	if bc.db == nil {
		return results, fmt.Errorf("database not available")
	}
	err := bc.db.View(func(tx *bbolt.Tx) error {
		foreignBucket := tx.Bucket([]byte(bc.bucketName + "_foreign"))
		if foreignBucket == nil {
			return fmt.Errorf("foreign index bucket not found")
		}

		// Get list of keys
		keysData := foreignBucket.Get([]byte(foreignKey))
		if keysData == nil {
			return nil // No items with this foreign key
		}

		var keys []string
		if err := json.Unmarshal(keysData, &keys); err != nil {
			return err
		}

		// Fetch each item
		bucket := tx.Bucket([]byte(bc.bucketName))
		if bucket == nil {
			return fmt.Errorf("bucket not found")
		}

		for _, key := range keys {
			data := bucket.Get([]byte(key))
			if data == nil {
				continue // Skip missing keys
			}

			var envelope recordEnvelope
			if err := json.Unmarshal(data, &envelope); err != nil {
				continue // Skip malformed data
			}

			item, err := decodeRecord(envelope)
			if err != nil {
				continue // Skip deserialization errors
			}

			results = append(results, &pkgcache.Item{Object: item})
		}

		return nil
	})

	return results, err
}

// GetKeys returns all primary keys in the cache
func (bc *boltStore) GetKeys() []string {
	keys := []string{}
	if bc.db == nil {
		return keys
	}
	bc.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(bc.bucketName))
		if bucket == nil {
			return nil
		}

		c := bucket.Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			keys = append(keys, string(k))
		}
		return nil
	})

	return keys
}

// GetForeignKeys returns all foreign keys in the cache
func (bc *boltStore) GetForeignKeys() []string {
	keys := []string{}
	bc.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(bc.bucketName + "_foreign"))
		if bucket == nil {
			return nil
		}

		c := bucket.Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			keys = append(keys, string(k))
		}
		return nil
	})

	return keys
}

// HasItemChanged checks if the item has changed by comparing hashes
func (bc *boltStore) HasItemChanged(key string, data interface{}) (bool, error) {
	newHash, _ := util.ComputeHash(data)

	var changed bool
	err := bc.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(bc.bucketName))
		if bucket == nil {
			return fmt.Errorf("bucket not found")
		}

		existingData := bucket.Get([]byte(key))
		if existingData == nil {
			// Item doesn't exist, so it's "changed"
			changed = true
			return nil
		}

		var envelope recordEnvelope
		if err := json.Unmarshal(existingData, &envelope); err != nil {
			return err
		}

		changed = envelope.Hash != newHash
		return nil
	})

	return changed, err
}

// HasItemBySecondaryKeyChanged checks if the item has changed using secondary key
func (bc *boltStore) HasItemBySecondaryKeyChanged(secondaryKey string, data interface{}) (bool, error) {
	var primaryKey string
	err := bc.db.View(func(tx *bbolt.Tx) error {
		secondaryBucket := tx.Bucket([]byte(bc.bucketName + "_secondary"))
		if secondaryBucket == nil {
			return errors.New("secondary index bucket not found")
		}

		pk := secondaryBucket.Get([]byte(secondaryKey))
		if pk == nil {
			return errors.New("secondary key not found")
		}
		primaryKey = string(pk)
		return nil
	})

	if err != nil {
		return true, err // Treat as changed if lookup fails
	}

	return bc.HasItemChanged(primaryKey, data)
}

// Delete removes an item by primary key
func (bc *boltStore) Delete(key string) error {
	if err := bc.ensureWritable(); err != nil {
		return err
	}
	return bc.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(bc.bucketName))
		if bucket == nil {
			return fmt.Errorf("bucket not found")
		}

		// Get envelope to find secondary keys and foreign key
		existingData := bucket.Get([]byte(key))
		if existingData == nil {
			return fmt.Errorf("key not found: %s", key)
		}

		var envelope recordEnvelope
		if err := json.Unmarshal(existingData, &envelope); err == nil {
			// Remove all secondary keys
			if len(envelope.Aliases) > 0 {
				secondaryBucket := tx.Bucket([]byte(bc.bucketName + "_secondary"))
				if secondaryBucket != nil {
					for secKey := range envelope.Aliases {
						secondaryBucket.Delete([]byte(secKey))
					}
				}
			}

			// Remove from foreign key index
			if envelope.GroupKey != "" {
				foreignBucket := tx.Bucket([]byte(bc.bucketName + "_foreign"))
				if foreignBucket != nil {
					keysData := foreignBucket.Get([]byte(envelope.GroupKey))
					if keysData != nil {
						var keys []string
						if err := json.Unmarshal(keysData, &keys); err == nil {
							// Remove this key from the list
							newKeys := []string{}
							for _, k := range keys {
								if k != key {
									newKeys = append(newKeys, k)
								}
							}

							if len(newKeys) > 0 {
								newKeysData, _ := json.Marshal(newKeys)
								foreignBucket.Put([]byte(envelope.GroupKey), newKeysData)
							} else {
								foreignBucket.Delete([]byte(envelope.GroupKey))
							}
						}
					}
				}
			}
		}

		// Delete from primary bucket
		return bucket.Delete([]byte(key))
	})
}

// DeleteBySecondaryKey removes an item by secondary key
func (bc *boltStore) DeleteBySecondaryKey(secondaryKey string) error {
	if bc.db == nil {
		return fmt.Errorf("database not available")
	}
	var primaryKey string
	err := bc.db.View(func(tx *bbolt.Tx) error {
		secondaryBucket := tx.Bucket([]byte(bc.bucketName + "_secondary"))
		if secondaryBucket == nil {
			return nil
		}

		pk := secondaryBucket.Get([]byte(secondaryKey))
		if pk != nil {
			primaryKey = string(pk)
		}
		return nil
	})

	if err != nil {
		return err
	}

	if primaryKey != "" {
		return bc.Delete(primaryKey)
	}

	return fmt.Errorf("secondary key not found: %s", secondaryKey)
}

// DeleteSecondaryKey removes a secondary key mapping
func (bc *boltStore) DeleteSecondaryKey(secondaryKey string) error {
	if err := bc.ensureWritable(); err != nil {
		return err
	}
	return bc.db.Update(func(tx *bbolt.Tx) error {
		secondaryBucket := tx.Bucket([]byte(bc.bucketName + "_secondary"))
		if secondaryBucket == nil {
			return nil
		}

		// Get primary key
		primaryKey := secondaryBucket.Get([]byte(secondaryKey))
		if primaryKey == nil {
			return nil
		}

		// Remove from secondary index
		if err := secondaryBucket.Delete([]byte(secondaryKey)); err != nil {
			return err
		}

		// Update envelope to remove this secondary key
		bucket := tx.Bucket([]byte(bc.bucketName))
		if bucket != nil {
			data := bucket.Get(primaryKey)
			if data != nil {
				var envelope recordEnvelope
				if err := json.Unmarshal(data, &envelope); err == nil {
					delete(envelope.Aliases, secondaryKey)
					envelopeData, _ := json.Marshal(envelope)
					bucket.Put(primaryKey, envelopeData)
				}
			}
		}

		return nil
	})
}

// DeleteForeignKey removes a foreign key from an item
func (bc *boltStore) DeleteForeignKey(foreignKey string) error {
	// This method removes the foreign key from an item, not all items with that key
	// Based on the cache.go implementation, this seems to be unused
	return nil
}

// DeleteItemsByForeignKey removes all items with the given foreign key
func (bc *boltStore) DeleteItemsByForeignKey(foreignKey string) error {
	if err := bc.ensureWritable(); err != nil {
		return err
	}
	return bc.db.Update(func(tx *bbolt.Tx) error {
		foreignBucket := tx.Bucket([]byte(bc.bucketName + "_foreign"))
		if foreignBucket == nil {
			return nil
		}

		// Get list of keys
		keysData := foreignBucket.Get([]byte(foreignKey))
		if keysData == nil {
			return nil
		}

		var keys []string
		if err := json.Unmarshal(keysData, &keys); err != nil {
			return err
		}

		// Delete each item
		bucket := tx.Bucket([]byte(bc.bucketName))
		if bucket == nil {
			return nil
		}

		for _, key := range keys {
			// We can't call Delete() here as we're in a transaction
			// So inline the deletion logic
			existingData := bucket.Get([]byte(key))
			if existingData != nil {
				var envelope recordEnvelope
				if err := json.Unmarshal(existingData, &envelope); err == nil {
					// Remove secondary keys
					if len(envelope.Aliases) > 0 {
						secondaryBucket := tx.Bucket([]byte(bc.bucketName + "_secondary"))
						if secondaryBucket != nil {
							for secKey := range envelope.Aliases {
								secondaryBucket.Delete([]byte(secKey))
							}
						}
					}
				}
				bucket.Delete([]byte(key))
			}
		}

		// Remove foreign key index entry
		return foreignBucket.Delete([]byte(foreignKey))
	})
}

// Flush removes all data from the cache.
// In read-only mode, this operation is silently skipped (no-op).
// Any errors during the flush operation are logged but not returned.
func (bc *boltStore) Flush() {
	// Skip flush in read-only mode
	if bc.db == nil || bc.db.IsReadOnly() {
		return
	}

	err := bc.db.Update(func(tx *bbolt.Tx) error {
		// Delete and recreate all buckets
		buckets := []string{
			bc.bucketName,
			bc.bucketName + "_secondary",
			bc.bucketName + "_foreign",
		}

		for _, bucketName := range buckets {
			if err := tx.DeleteBucket([]byte(bucketName)); err != nil && err != bbolt.ErrBucketNotFound {
				return err
			}
			if _, err := tx.CreateBucket([]byte(bucketName)); err != nil {
				return err
			}
		}

		return nil
	})

	// Log error if flush fails (but don't panic or propagate it)
	if err != nil {
		// Note: We can't use a logger here as we don't have access to log.FieldLogger
		// in this type. Errors would need to be logged at the manager level.
		// The cache.Cache interface doesn't support returning errors from Flush()
		fmt.Fprintf(os.Stderr, "WARNING: failed to flush cache %s: %v\n", bc.bucketName, err)
	}
}

// Save is a no-op for bbolt persistence.
//
// Unlike in-memory caches that require explicit serialization, bbolt automatically persists
// all data to disk as part of the transaction. Calling this method has no effect.
// This method exists to satisfy the cache.Cache interface contract for compatibility
// with other cache implementations.
func (bc *boltStore) Save(path string) error {
	return nil
}

// Load is a no-op for bbolt persistence.
//
// Unlike in-memory caches that require explicit deserialization from disk, bbolt
// automatically loads all data from the database file on open. Data is available
// immediately and does not require a separate load operation.
// This method exists to satisfy the cache.Cache interface contract for compatibility
// with other cache implementations.
func (bc *boltStore) Load(path string) error {
	return nil
}

// GetItem - bbolt doesn't expose Item metadata externally
// This returns the raw object, not wrapped in cache.Item
func (bc *boltStore) GetItem(key string) (*pkgcache.Item, error) {
	obj, err := bc.Get(key)
	if err != nil {
		return nil, err
	}
	// Return a minimal Item wrapper for compatibility
	// Note: bbolt stores metadata internally, so we don't expose full Item details
	return &pkgcache.Item{Object: obj}, nil
}

// GetItemBySecondaryKey - bbolt doesn't expose Item metadata externally
func (bc *boltStore) GetItemBySecondaryKey(secondaryKey string) (*pkgcache.Item, error) {
	obj, err := bc.GetBySecondaryKey(secondaryKey)
	if err != nil {
		return nil, err
	}
	return &pkgcache.Item{Object: obj}, nil
}
