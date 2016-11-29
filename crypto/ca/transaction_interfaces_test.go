package ca

// Concrete implementations of Issuance, Spend, and Output interfaces
// useful for testing.

type issuanceStruct struct {
	AD   AssetDescriptor
	VD   ValueDescriptor
	AIDs []AssetID
	IARP *IssuanceAssetRangeProof
	VRP  *ValueRangeProof
}

func (i issuanceStruct) AssetDescriptor() AssetDescriptor                  { return i.AD }
func (i issuanceStruct) ValueDescriptor() ValueDescriptor                  { return i.VD }
func (i issuanceStruct) AssetIDs() []AssetID                               { return i.AIDs }
func (i issuanceStruct) IssuanceAssetRangeProof() *IssuanceAssetRangeProof { return i.IARP }
func (i issuanceStruct) ValueRangeProof() *ValueRangeProof                 { return i.VRP }

type inputStruct struct {
	AD AssetDescriptor
	VD ValueDescriptor
}

func (in inputStruct) AssetDescriptor() AssetDescriptor { return in.AD }
func (in inputStruct) ValueDescriptor() ValueDescriptor { return in.VD }

type outputStruct struct {
	AD  AssetDescriptor
	VD  ValueDescriptor
	ARP *AssetRangeProof
	VRP *ValueRangeProof
}

func (o outputStruct) AssetDescriptor() AssetDescriptor  { return o.AD }
func (o outputStruct) ValueDescriptor() ValueDescriptor  { return o.VD }
func (o outputStruct) AssetRangeProof() *AssetRangeProof { return o.ARP }
func (o outputStruct) ValueRangeProof() *ValueRangeProof { return o.VRP }
