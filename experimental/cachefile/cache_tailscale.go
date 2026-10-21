package cachefile

import "github.com/sagernet/bbolt"

var (
	bucketTailscaleForceRelay = []byte("ts_force_relay")
	keyTailscaleForceRelay    = []byte("enabled")
)

func (c *CacheFile) LoadTailscaleForceRelay() (enabled bool, loaded bool, err error) {
	err = c.view(func(t *bbolt.Tx) error {
		bucket := c.bucket(t, bucketTailscaleForceRelay)
		if bucket == nil {
			return nil
		}
		value := bucket.Get(keyTailscaleForceRelay)
		if len(value) == 1 {
			enabled = value[0] == 1
			loaded = true
		}
		return nil
	})
	return
}

func (c *CacheFile) StoreTailscaleForceRelay(enabled bool) error {
	return c.batch(func(t *bbolt.Tx) error {
		bucket, err := c.createBucket(t, bucketTailscaleForceRelay)
		if err != nil {
			return err
		}
		value := byte(0)
		if enabled {
			value = 1
		}
		return bucket.Put(keyTailscaleForceRelay, []byte{value})
	})
}
