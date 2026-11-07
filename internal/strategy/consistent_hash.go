package strategy

import (
	"errors"
	"hash/crc32"
	"sort"
	"strconv"
	"sync"

	"github.com/nikhil-thorat/relay/internal/target"
)

const virtualNodes = 128

type hashNode struct {
	hash uint32
	id   string
}

type ConsistentHash struct {
	mu      sync.Mutex
	counter int
	ringIDs []string
	ring    []hashNode
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

func (c *ConsistentHash) buildRingLocked(targets []target.Target) {
	if len(c.ringIDs) == len(targets) {
		same := true
		for i := range targets {
			if c.ringIDs[i] != targets[i].ID {
				same = false
				break
			}
		}
		if same {
			return
		}
	}

	c.ringIDs = make([]string, len(targets))
	c.ring = c.ring[:0]

	for i, t := range targets {
		c.ringIDs[i] = t.ID
		for j := range virtualNodes {
			c.ring = append(c.ring, hashNode{
				hash: hashKey(t.ID + "#" + strconv.Itoa(j)),
				id:   t.ID,
			})
		}
	}

	sort.Slice(c.ring, func(i, j int) bool {
		return c.ring[i].hash < c.ring[j].hash
	})
}

func (c *ConsistentHash) Prune(_ []target.Target) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.ringIDs = nil
	c.ring = c.ring[:0]
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

	c.mu.Lock()
	defer c.mu.Unlock()

	c.buildRingLocked(targets)

	h := hashKey(key)
	idx := sort.Search(len(c.ring), func(i int) bool {
		return c.ring[i].hash >= h
	})
	if idx == len(c.ring) {
		idx = 0
	}

	id := c.ring[idx].id
	for _, t := range targets {
		if t.ID == id {
			return t, nil
		}
	}

	return target.Target{}, errors.New("no target to select from")
}
