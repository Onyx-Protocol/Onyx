package raft

import "context"

const potentialMemberPrefix = "potential"

// AddPotentialMember adds an address for a potential new member to the
// list of potential cluster members.
// An address must be listed as a potential cluster member before the node
// listening on that address can join the cluster.
func (sv *Service) AddPotentialMember(ctx context.Context, addr string) error {
	dummyData := []byte{0x01}
	return sv.Set(ctx, potentialMemberPrefix+"/"+addr, dummyData)
}

func (sv *Service) isPotentialMember(ctx context.Context, addr string) bool {
	data, err := sv.Get(ctx, potentialMemberPrefix+"/"+addr)
	if err != nil {
		return false
	}
	return len(data) > 0
}
