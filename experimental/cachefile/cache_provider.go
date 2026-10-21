package cachefile

import (
	"os"

	"github.com/sagernet/bbolt"
	"github.com/sagernet/sing-box/adapter"
)

var bucketOutboundProvider = []byte("outbound_provider")

func (c *CacheFile) LoadOutboundProvider(tag string) *adapter.SavedOutboundProvider {
	var saved adapter.SavedOutboundProvider
	err := c.view(func(t *bbolt.Tx) error {
		bucket := c.bucket(t, bucketOutboundProvider)
		if bucket == nil {
			return os.ErrNotExist
		}
		content := bucket.Get([]byte(tag))
		if len(content) == 0 {
			return os.ErrInvalid
		}
		return saved.UnmarshalBinary(content)
	})
	if err != nil {
		return nil
	}
	return &saved
}

func (c *CacheFile) SaveOutboundProvider(tag string, provider *adapter.SavedOutboundProvider) error {
	return c.batch(func(t *bbolt.Tx) error {
		bucket, err := c.createBucket(t, bucketOutboundProvider)
		if err != nil {
			return err
		}
		content, err := provider.MarshalBinary()
		if err != nil {
			return err
		}
		return bucket.Put([]byte(tag), content)
	})
}
