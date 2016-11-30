package core

import (
	stdcontext "context"

	"golang.org/x/net/context"

	"chain/core/mockhsm"
	"chain/core/pb"
	"chain/core/txbuilder"
	"chain/crypto/ed25519/chainkd"
)

func (h *Handler) CreateKey(ctx context.Context, in *pb.CreateKeyRequest) (*pb.CreateKeyResponse, error) {
	result, err := h.HSM.XCreate(ctx, in.Alias)
	if err != nil {
		return nil, err
	}
	xpub := &pb.XPub{Xpub: result.XPub[:]}
	if result.Alias != nil {
		xpub.Alias = *result.Alias
	}
	return &pb.CreateKeyResponse{Xpub: xpub}, nil
}

func (h *Handler) ListKeys(ctx context.Context, in *pb.ListKeysQuery) (*pb.ListKeysResponse, error) {
	limit := in.PageSize
	if limit == 0 {
		limit = defGenericPageSize
	}

	xpubs, after, err := h.HSM.ListKeys(ctx, in.Aliases, in.After, int(limit))
	if err != nil {
		return nil, err
	}

	var items []*pb.XPub
	for _, xpub := range xpubs {
		proto := &pb.XPub{Xpub: xpub.XPub[:]}
		if xpub.Alias != nil {
			proto.Alias = *xpub.Alias
		}
		items = append(items, proto)
	}

	in.After = after

	return &pb.ListKeysResponse{
		Items:    items,
		LastPage: len(xpubs) < int(limit),
		Next:     in,
	}, nil
}

func (h *Handler) DeleteKey(ctx context.Context, in *pb.DeleteKeyRequest) (*pb.ErrorResponse, error) {
	var key chainkd.XPub
	if len(in.Xpub) != len(key) {
		return nil, chainkd.ErrBadKeyLen
	}
	copy(key[:], in.Xpub)
	return nil, h.HSM.DeleteChainKDKey(ctx, key)
}

func (h *Handler) SignTxs(ctx context.Context, in *pb.SignTxsRequest) (*pb.TxsResponse, error) {
	xpubs, err := bytesToKeys(in.Xpubs)
	if err != nil {
		return nil, err
	}

	responses := make([]*pb.TxsResponse_Response, len(in.Transactions))
	for i, tx := range in.Transactions {
		err := txbuilder.Sign(ctx, tx, xpubs, h.mockhsmSignTemplate)
		if err != nil {
			responses[i] = &pb.TxsResponse_Response{Error: protobufErr(err)}
		} else {
			responses[i] = &pb.TxsResponse_Response{Template: tx}
		}
	}
	return &pb.TxsResponse{Responses: responses}, nil
}

func (h *Handler) mockhsmSignTemplate(ctx stdcontext.Context, xpub chainkd.XPub, path [][]byte, data [32]byte) ([]byte, error) {
	sigBytes, err := h.HSM.XSign(ctx, xpub, path, data[:])
	if err == mockhsm.ErrNoKey {
		return nil, nil
	}
	return sigBytes, err
}
