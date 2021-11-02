package cache

// Item - a cached item
type Item struct {
	Object        interface{}     `json:"data"`
	UpdateTime    int64           `json:"updateTime"`
	Hash          uint64          `json:"hash"`
	SecondaryKeys map[string]bool `json:"secondaryKeys"` // keep track of secondary keys for clean up
	ForeignKey    string          `json:"foreignKey"`
}

// GetObject - returns the object saved in this cache item
func (i *Item) GetObject() interface{} {
	return i.Object
}

// GetUpdateTime - returns the epoch time that this cache item was updated
func (i *Item) GetUpdateTime() int64 {
	return i.UpdateTime
}

// GetHash - returns the hash of the object in this cache item
func (i *Item) GetHash() uint64 {
	return i.Hash
}
