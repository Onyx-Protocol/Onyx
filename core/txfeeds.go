package core

import (
	"fmt"
	"math"

	"golang.org/x/net/context"

	"chain/core/pb"
	"chain/core/query"
	"chain/core/txfeed"
	"chain/errors"
	"chain/net/http/httpjson"
)

func (h *Handler) CreateTxFeed(ctx context.Context, in *pb.CreateTxFeedRequest) (*pb.TxFeedResponse, error) {
	after := fmt.Sprintf("%d:%d-%d", h.Chain.Height(), math.MaxInt32, uint64(math.MaxInt64))
	feed, err := h.TxFeeds.Create(ctx, in.Alias, in.Filter, after, in.ClientToken)
	if err != nil {
		return nil, err
	}
	return &pb.TxFeedResponse{Response: txFeedProto(feed)}, nil
}

func (h *Handler) GetTxFeed(ctx context.Context, in *pb.GetTxFeedRequest) (*pb.TxFeedResponse, error) {
	feed, err := h.TxFeeds.Find(ctx, in.GetId(), in.GetAlias())
	if err != nil {
		return nil, err
	}
	return &pb.TxFeedResponse{Response: txFeedProto(feed)}, nil
}

func (h *Handler) DeleteTxFeed(ctx context.Context, in *pb.DeleteTxFeedRequest) (*pb.ErrorResponse, error) {
	err := h.TxFeeds.Delete(ctx, in.GetId(), in.GetAlias())
	return nil, err
}

func (h *Handler) UpdateTxFeed(ctx context.Context, in *pb.UpdateTxFeedRequest) (*pb.TxFeedResponse, error) {
	// TODO(tessr): Consider moving this function into the txfeed package.
	// (It's currently outside the txfeed package to avoid a dependecy cycle
	// between txfeed and query.)
	bad, err := txAfterIsBefore(in.After, in.PreviousAfter)
	if err != nil {
		return nil, err
	}

	if bad {
		return nil, errors.WithDetail(httpjson.ErrBadRequest, "new After cannot be before Prev")
	}

	feed, err := h.TxFeeds.Update(ctx, in.GetId(), in.GetAlias(), in.After, in.PreviousAfter)
	if err != nil {
		return nil, err
	}
	return &pb.TxFeedResponse{Response: txFeedProto(feed)}, nil
}

// txAfterIsBefore returns true if a is before b. It returns an error if either
// a or b are not valid query.TxAfters.
func txAfterIsBefore(a, b string) (bool, error) {
	aAfter, err := query.DecodeTxAfter(a)
	if err != nil {
		return false, err
	}

	bAfter, err := query.DecodeTxAfter(b)
	if err != nil {
		return false, err
	}

	return aAfter.FromBlockHeight < bAfter.FromBlockHeight ||
		(aAfter.FromBlockHeight == bAfter.FromBlockHeight &&
			aAfter.FromPosition < bAfter.FromPosition), nil
}

func txFeedProto(f *txfeed.TxFeed) *pb.TxFeed {
	proto := &pb.TxFeed{
		Id:     f.ID,
		Filter: f.Filter,
		After:  f.After,
	}
	if f.Alias != nil {
		proto.Alias = *f.Alias
	}
	return proto
}
