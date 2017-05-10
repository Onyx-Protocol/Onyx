package ivy

type typeDesc string

var (
	amountType      = typeDesc("Amount")
	assetType       = typeDesc("Asset")
	assetAmountType = typeDesc("AssetAmount")
	boolType        = typeDesc("Boolean")
	contractType    = typeDesc("Contract")
	hashType        = typeDesc("Hash")
	intType         = typeDesc("Integer")
	listType        = typeDesc("List")
	nilType         = typeDesc("")
	predType        = typeDesc("Predicate")
	progType        = typeDesc("Program")
	pubkeyType      = typeDesc("PublicKey")
	sigType         = typeDesc("Signature")
	strType         = typeDesc("String")
	timeType        = typeDesc("Time")
	valueType       = typeDesc("Value")
)

var types = map[string]typeDesc{
	string(amountType):      amountType,
	string(assetType):       assetType,
	string(assetAmountType): assetAmountType,
	string(boolType):        boolType,
	string(hashType):        hashType,
	string(intType):         intType,
	string(listType):        listType,
	string(nilType):         nilType,
	string(predType):        predType,
	string(progType):        progType,
	string(pubkeyType):      pubkeyType,
	string(sigType):         sigType,
	string(strType):         strType,
	string(timeType):        timeType,
	string(valueType):       valueType,
}
