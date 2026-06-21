package strategy

import (
	"fmt"
)

func New(name string) (Strategy, error) {
	switch name {
	case "round_robin":
		return &RoundRobin{}, nil
	default:
		return nil, fmt.Errorf("unkown strategy : %s", name)
	}
}
