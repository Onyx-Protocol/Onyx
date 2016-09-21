package filter

type binaryOp struct {
	precedence int
	name       string // AND, =, etc.
}

var binaryOps = map[string]*binaryOp{
	"OR":  {1, "OR"},
	"AND": {2, "AND"},
	"=":   {3, "="},
}
