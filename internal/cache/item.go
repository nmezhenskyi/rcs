package cache

import "time"

// item represents a value stored in cache, it may have an expiration date.
type item struct {
	data    []byte
	expires int64 // if zero, item never expires.
}

func (i *item) isExpired() bool {
	if i.expires == 0 {
		return false
	}
	return time.Now().UnixNano() > i.expires
}
