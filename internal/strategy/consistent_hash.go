package strategy

import (
	"errors"
	"hash/crc32"
	"sort"
	"sync"

	"github.com/nikhil-thorat/relay/internal/target"
)

type ConsistentHash struct {
	mu      sync.Mutex
	counter int
}

func hashKey(key string) uint32 {
	return crc32.ChecksumIEEE([]byte(key))
}

func sortedCopy(targets []target.Target) []target.Target {
	out := make([]target.Target, len(targets))
	copy(out, targets)
	sort.Slice(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out
}

func (c *ConsistentHash) Select(targets []target.Target) (target.Target, error) {
	if len(targets) == 0 {
		return target.Target{}, errors.New("no target to select from")
	}

	if len(targets) == 1 {
		return targets[0], nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	ordered := sortedCopy(targets)
	selected := ordered[c.counter%len(ordered)]
	c.counter++

	return selected, nil
}

func (c *ConsistentHash) SelectWithKey(targets []target.Target, key string) (target.Target, error) {
	if len(targets) == 0 {
		return target.Target{}, errors.New("no target to select from")
	}

	if len(targets) == 1 {
		return targets[0], nil
	}

	if key == "" {
		return c.Select(targets)
	}

	ordered := sortedCopy(targets)
	idx := int(hashKey(key) % uint32(len(ordered)))

	return ordered[idx], nil
}
