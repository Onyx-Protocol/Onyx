package core

import (
	"sync"

	"golang.org/x/net/context"

	"chain/core/pb"
	"chain/net/http/reqid"
)

func (h *Handler) CreateControlPrograms(ctx context.Context, in *pb.CreateControlProgramsRequest) (*pb.CreateControlProgramsResponse, error) {
	responses := make([]*pb.CreateControlProgramsResponse_Response, len(in.Requests))
	var wg sync.WaitGroup
	wg.Add(len(responses))

	for i := 0; i < len(responses); i++ {
		go func(i int) {
			ctx = reqid.NewSubContext(ctx, reqid.New())
			defer wg.Done()
			defer batchRecover(func(err error) {
				responses[i] = &pb.CreateControlProgramsResponse_Response{Error: protobufErr(err)}
			})
			switch in.Requests[i].GetType().(type) {
			case (*pb.CreateControlProgramsRequest_Request_Account):
				responses[i] = h.createAccountControlProgram(ctx, in.Requests[i].GetAccount())
			default:
				responses[i] = &pb.CreateControlProgramsResponse_Response{Error: protobufErr(errBadAction)}
			}
		}(i)
	}

	wg.Wait()
	return &pb.CreateControlProgramsResponse{Responses: responses}, nil
}

func (h *Handler) createAccountControlProgram(ctx context.Context, in *pb.CreateControlProgramsRequest_Account) *pb.CreateControlProgramsResponse_Response {
	resp := new(pb.CreateControlProgramsResponse_Response)

	accountID := in.GetAccountId()
	if accountID == "" {
		acc, err := h.Accounts.FindByAlias(ctx, in.GetAccountAlias())
		if err != nil {
			resp.Error = protobufErr(err)
			return resp
		}
		accountID = acc.ID
	}

	controlProgram, err := h.Accounts.CreateControlProgram(ctx, accountID, false)
	if err != nil {
		resp.Error = protobufErr(err)
		return resp
	}

	resp.ControlProgram = controlProgram
	return resp
}
