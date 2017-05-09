{
let BugError = require("./errors").BugError

let ast = require("./ast")

let types = require("./cvm/types")
let instructions = require("./cvm/instructions")

let createInstructionExpression = ast.createInstructionExpression
let createBinaryExpression = ast.createBinaryExpression

let isDeclarableType = types.isDeclarableType
let isDeclarableUnaryOperator = instructions.isDeclarableUnaryOperator
let isComparisonOperator = instructions.isComparisonOperator
let isArithmeticOperator = instructions.isArithmeticOperator

}

Contract
  = __ "contract" _ name:Identifier "(" __ parameters:Parameters __ ")" __ "{" __ clauses:Clause+ "}" __ { return { type: "rawContract", location: location(), name: name, parameters: parameters, clauses: clauses} }

Clause
  = "clause" _ name:Identifier "(" __ parameters:Parameters __ ")" __ "{" __ assertions:Assertion* __ outputs:Output*  __  returnStatement: Return? "}" __ { 
    return { 
      type: "clause", 
      location: location(), 
      name: name, 
      parameters: parameters, 
      assertions: assertions,
      outputs: outputs,
      returnStatement: returnStatement
    } 
  }

Assertion
  = "verify" _ exp:Expression1 __ { return { type: "assertion", location: location(), expression: exp} }

Output
  = "output" _ contract:ContractExpression __ { return { type: "output", location: location(), contract: contract } }

ContractExpression
  = address:VariableExpression "(" value:VariableExpression ")" { return { type: "contractExpression", location: location(), address: address, value: value } }

Return
  = "return" _ value:VariableExpression __ { return { type: "returnStatement", location: location(), value: value } }

// need to handle precedence

Expression1 "expression"
  = ComparisonExpression
  / Expression2

Expression2
  = BinaryExpression
  / Expression3

Expression3
  = UnaryExpression
  / Expression4

Expression4
  = CallExpression
  / Literal
  / VariableExpression
  / "(" exp:Expression1 ")" { return exp }

Literal
  = ListLiteral
  / IntegerLiteral
  / BooleanLiteral

ComparisonExpression // not associative
  = left:Expression2 __ operator:ComparisonOperator __ right:Expression2 { return createBinaryExpression([{left: left, operator: operator}], right) }

ComparisonOperator
  = (operator:Operator & { return isComparisonOperator(operator) }) { return text() }

BinaryExpression // left associative
  = partials:PartialBinaryExpression+ right:Expression3 { return createBinaryExpression(partials, right) }

PartialBinaryExpression
  = left:Expression3 __ operator:BinaryOperator __ { return { type: "partial", location: location(), left: left, operator: operator } }

BinaryOperator
  = (operator:Operator & { return isArithmeticOperator(operator) }) { return text() }

CallExpression
  = name:FunctionIdentifier "(" args:Expressions ")" { return createInstructionExpression("callExpression", location(), name, args) }

UnaryExpression
  = operator:Operator arg:Expression4 { return createInstructionExpression("unaryExpression", location(), operator, [arg]) }

UnaryOperator
  = (operator:Operator & { isDeclarableUnaryOperator(operator) }) { return text() }

VariableExpression
  = identifier:VariableName { return { type: "variable", location: location(), identifier: identifier} }

VariableName
  = (Identifier "." Identifier) { return text() }
  / Identifier

Expressions "expressions"
  = first:Expression1 "," __ rest:Expressions { rest.unshift(first); return rest }
  / exp:Expression1 { return [exp] }
  / Nothing

ListLiteral "listLiteral"
  = "[" values:Expressions "]" { return { type: "listLiteral", location: location(), text: text(), values: values } }

IntegerLiteral "integer"
  = [-]?[0-9]+ { return { type: "literal", literalType: "Number", location: location(), value: text() } }

BooleanLiteral "boolean"
  = ("true" / "false") { return { type: "literal", literalType: "Boolean", location: location(), value: text() } }

Identifiers "identifiers"
  = first:Identifier "," __ rest:Identifiers { rest.unshift(first); return rest }
  / Identifier { return [text()] }
  / Nothing

Parameters "parameters"
  = first:Parameter "," __ rest:Parameters { rest.unshift(first); return rest }
  / param:Parameter { return [param] }
  / Nothing

Parameter "parameter"
  = id:Identifier ": " type:Type { return { type: "parameter", location: location(), itemType: type, identifier: id } }

Type "type"
  = (id:Identifier & { return isDeclarableType(id) }) { return text() }

Identifier "identifier"
  = [_A-Za-z] [_A-Za-z0-9]* { return text() }

FunctionIdentifier "functionIdentifier"
  = Identifier "." Identifier { return text() }
  / Identifier

Nothing "nothing"
  = __ { return [] }

__ "optional whitespace"
  = _?

_ "whitespace"
  = [ \t\n\r;]+ (Comment)? __
  / Comment __

Comment "comment"
  = "//" [^\n\r]* /
    "/*" (!"*/" .)* "*/"

Operator
  = [^ \t\n\rA-Za-z1-9\(\)]+ { return text() }
