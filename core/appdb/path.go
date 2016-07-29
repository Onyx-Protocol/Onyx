package appdb

const (
	CustomerPaymentNamespace = 0
	CustomerAssetsNamespace  = 1
)

func IssuancePath(asset *Asset) []uint32 {
	return []uint32{
		CustomerAssetsNamespace,
		asset.INIndex[0],
		asset.INIndex[1],
		asset.AIndex[0],
		asset.AIndex[1],
	}
}
