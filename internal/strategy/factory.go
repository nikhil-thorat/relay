package strategy

import (
	"fmt"
)

func New(name string) (Strategy, error) {
	switch name {
	case "round_robin":
		return &RoundRobin{}, nil
	case "weighted_round_robin":
		return &WeightedRoundRobin{}, nil
	case "least_conn":
		return &LeastConnections{}, nil
	case "least_response_time":
		return &LeastResponseTime{}, nil
	case "consistent_hash":
		return &ConsistentHash{}, nil
	default:
		return nil, fmt.Errorf("unknown strategy: %s", name)
	}
}
