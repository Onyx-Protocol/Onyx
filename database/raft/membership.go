package raft

import "context"

const allowedMemberPrefix = "/raft/allowed"

// AddAllowedMember adds an address for a member to the list of allowed cluster members.
// An address must be listed as a allowed cluster member before the node
// listening on that address can join the cluster.
func (sv *Service) AddAllowedMember(ctx context.Context, addr string) error {
	dummyData := []byte{0x01}
	return sv.Set(ctx, allowedMemberPrefix+"/"+addr, dummyData)
}

func (sv *Service) isAllowedMember(ctx context.Context, addr string) bool {
	data, err := sv.Get(ctx, allowedMemberPrefix+"/"+addr)
	if err != nil {
		return false
	}
	return len(data) > 0
}
