package filter

type binaryOp struct {
	precedence int
	name       string // AND, =, etc.
	sqlOp      string
}

var binaryOps = map[string]*binaryOp{
	"OR":  {1, "OR", "OR"},
	"AND": {2, "AND", "AND"},
	"=":   {3, "=", "="},
}
