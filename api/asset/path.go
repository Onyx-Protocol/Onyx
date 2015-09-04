package asset

import "chain/api/appdb"

const (
	CustomerPaymentNamespace = 0
	CustomerAssetsNamespace  = 1
)

func ReceiverPath(addr *appdb.Address) []uint32 {
	return []uint32{
		CustomerPaymentNamespace,
		addr.BucketIndex[0],
		addr.BucketIndex[1],
		addr.Index[0],
		addr.Index[1],
	}
}

func IssuancePath(asset *appdb.Asset) []uint32 {
	return []uint32{
		CustomerAssetsNamespace,
		asset.AGIndex[0],
		asset.AGIndex[1],
		asset.AIndex[0],
		asset.AIndex[1],
	}
}
