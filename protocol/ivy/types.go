package ivy

type typeDesc string

var (
	amountType           = typeDesc("Amount")
	assetType            = typeDesc("Asset")
	boolType             = typeDesc("Boolean")
	contractType         = typeDesc("Contract")
	intType              = typeDesc("Integer")
	listType             = typeDesc("List")
	nilType              = typeDesc("")
	predType             = typeDesc("Predicate")
	progType             = typeDesc("Program")
	pubkeyType           = typeDesc("PublicKey")
	sha256PubkeyHashType = typeDesc("Sha256PublicKeyHash")
	sha256StrHashType    = typeDesc("Sha256StringHash")
	sha3PubkeyHashType   = typeDesc("Sha3PublicKeyHash")
	sha3StrHashType      = typeDesc("Sha3StringHash")
	sigType              = typeDesc("Signature")
	strType              = typeDesc("String")
	timeType             = typeDesc("Time")
	valueType            = typeDesc("Value")
)

var types = map[string]typeDesc{
	string(amountType):           amountType,
	string(assetType):            assetType,
	string(boolType):             boolType,
	string(intType):              intType,
	string(listType):             listType,
	string(nilType):              nilType,
	string(predType):             predType,
	string(progType):             progType,
	string(pubkeyType):           pubkeyType,
	string(sha256PubkeyHashType): sha256PubkeyHashType,
	string(sha256StrHashType):    sha256StrHashType,
	string(sha3PubkeyHashType):   sha3PubkeyHashType,
	string(sha3StrHashType):      sha3StrHashType,
	string(sigType):              sigType,
	string(strType):              strType,
	string(timeType):             timeType,
	string(valueType):            valueType,
}
