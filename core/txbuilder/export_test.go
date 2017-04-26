package txbuilder

var CheckTxSighashCommitment = checkTxSighashCommitment

type SignatureWitness signatureWitness

func (sw SignatureWitness) materialize(args *[][]byte) error {
	return signatureWitness(sw).materialize(args)
}

func AsSignatureWitness(wc witnessComponent) *SignatureWitness {
	return (*SignatureWitness)(wc.(*signatureWitness))
}
