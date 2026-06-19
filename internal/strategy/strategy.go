package strategy

import "github.com/nikhil-thorat/relay/internal/target"

type Strategy interface {
	Select(targets []*target.Target) (*target.Target, error)
}
