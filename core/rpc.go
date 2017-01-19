package core

import (
	libcontext "golang.org/x/net/context"

	"chain/core/leader"
	"chain/core/pb"
	"chain/errors"
	"chain/protocol/bc"
)

func (h *Handler) GetBlock(ctx libcontext.Context, in *pb.GetBlockRequest) (*pb.GetBlockResponse, error) {
	err := <-h.Chain.BlockSoonWaiter(ctx, in.Height)
	if err != nil {
		return nil, errors.Wrapf(err, "waiting for block at height %d", in.Height)
	}

	rawBlock, err := h.Store.GetRawBlock(ctx, in.Height)
	if err != nil {
		return nil, err
	}

	return &pb.GetBlockResponse{Block: rawBlock}, nil
}

func (h *Handler) GetSnapshotInfo(ctx libcontext.Context, in *pb.Empty) (*pb.GetSnapshotInfoResponse, error) {
	height, size, err := h.Store.LatestSnapshotInfo(ctx)
	if err != nil {
		return nil, err
	}
	return &pb.GetSnapshotInfoResponse{
		Height:       height,
		Size:         size,
		BlockchainId: h.Config.BlockchainID[:],
	}, nil
}

func (h *Handler) GetSnapshot(ctx libcontext.Context, in *pb.GetSnapshotRequest) (*pb.GetSnapshotResponse, error) {
	data, err := h.Store.GetSnapshot(ctx, in.Height)
	if err != nil {
		return nil, err
	}

	return &pb.GetSnapshotResponse{Data: data}, nil
}

func (h *Handler) GetBlockHeight(ctx libcontext.Context, in *pb.Empty) (*pb.GetBlockHeightResponse, error) {
	return &pb.GetBlockHeightResponse{Height: h.Chain.Height()}, nil
}

func (h *Handler) SubmitTx(ctx libcontext.Context, in *pb.SubmitTxRequest) (*pb.SubmitTxResponse, error) {
	txdata, err := bc.NewTxDataFromBytes(in.Transaction)
	if err != nil {
		return nil, err
	}
	err = h.Submitter.Submit(ctx, bc.NewTx(*txdata))
	if err != nil {
		return nil, err
	}
	return &pb.SubmitTxResponse{Ok: true}, nil
}

func (h *Handler) SignBlock(ctx libcontext.Context, in *pb.SignBlockRequest) (*pb.SignBlockResponse, error) {
	if !leader.IsLeading() {
		conn, err := leaderConn(ctx, h.DB, h.Addr)
		if err != nil {
			return nil, err
		}
		defer conn.Conn.Close()

		return pb.NewSignerClient(conn.Conn).SignBlock(ctx, in)
	}
	block, err := bc.NewBlockFromBytes(in.Block)
	if err != nil {
		return nil, err
	}
	sig, err := h.Signer(ctx, block)
	if err != nil {
		return nil, err
	}
	return &pb.SignBlockResponse{Signature: sig}, nil
}
