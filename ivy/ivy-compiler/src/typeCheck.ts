import {
  TypeVariable,
  TypeSignature,
  Type,
  List,
  isList,
  TypeClass,
  typeToString,
  inputTypesToString,
  getTypeClass,
  isTypeVariable,
  isPrimitive,
  createTypeSignature,
  isHash,
  Hash,
  isTypeClass
} from './cvm/types'

import {
  RawContract,
  Clause,
  Expression,
  Statement,
  Assertion,
  Return,
  Output,
  ValueLiteral,
  scopedName,
  Parameter,
  Variable,
  mapOverAST,
  ListLiteral,
  ASTNode
} from './ast'

import {
  getTypeSignature
} from './cvm/instructions'

import {
  IvyTypeError,
  BugError,
  NameError
} from './errors'

export type TypeMap = Map<string, Type>

export type TypeConstraint = {
  type: "typeConstraint"
  left: Type, 
  right: Type
}

export type TypeClassConstraint = {
  type: "typeClassConstraint"
  left: TypeVariable, 
  right: TypeClass
}

function mapOverType(func: (TypeVariable) => Type, type: Type): Type {
  // call func on all type variables
  switch (getTypeClass(type)) {
    case "Primitive": return type
    case "Other": return type
    case "TypeVariable": return func(type)
    case "Hash": return { type: "hashType", hashFunction: (type as Hash).hashFunction, inputType: mapOverType(func, (type as Hash).inputType) }
    case "List": return { type: "listType", elementType: mapOverType(func, (type as List).elementType) }
  }
}

function mapOverConstraints(func: (TypeVariable) => Type, constraints: TypeConstraint[]): TypeConstraint[] {
  return constraints.map((constraint) => mapOverConstraint(func, constraint))
}

function mapOverConstraint(func: (TypeVariable) => Type, constraint: TypeConstraint): TypeConstraint {
  return { 
    type: "typeConstraint",
    left: mapOverType(func, constraint.left),
    right: mapOverType(func, constraint.right),
  }
}

function mapOverTypeMap(func: (TypeVariable) => Type, typeMap: TypeMap): TypeMap {
  let newMap = new Map<string, Type>()
  for (let [name, type] of typeMap) {
    newMap.set(name, mapOverType(func, type))
  }
  return newMap
}

function isSameType(type1: Type, type2: Type) {
  let typeClass = getTypeClass(type1)
  if (typeClass !== getTypeClass(type2)) return false
  switch (typeClass) {
    case "Primitive": 
    case "Other":
      return type1 === type2
    case "TypeVariable": return (type1 as TypeVariable).name === (type2 as TypeVariable).name
    case "Hash": return ((type1 as Hash).hashFunction === (type2 as Hash).hashFunction) && isSameType((type1 as Hash).inputType, (type2 as Hash).inputType)
    case "List": return isSameType((type1 as List).elementType, (type2 as List).elementType)
  }
}

function checkOccurs(targetTypeVariable: Type, type: Type) {
  // check if a typeVariable occurs in a type
  // if it does; throw an error
  mapOverType((typeVariable) => {
    if (isSameType(typeVariable, targetTypeVariable)) throw new IvyTypeError("infinitely recurrent type: " + typeToString(targetTypeVariable) + " in " + typeToString(type))
    return typeVariable
  }, type)
}

function isSimplified(type: Type) {
  let result = { simplified: true }
  mapOverType((typeVariable) => {
    result.simplified = false
    return typeVariable
  }, type)
  return result.simplified
}

function applyConstraints(constraints: TypeConstraint[], mapping: TypeMap = new Map<string, Type>()): TypeMap {
  let constraint = constraints.shift()
  if (constraint === undefined) return mapping
  let left = constraint.left
  let right = constraint.right
  return unify(left, right, constraints, mapping)
}

function applyTypeClassConstraints(constraints: TypeClassConstraint[], mapping: TypeMap): TypeMap {
  for (let constraint of constraints) {
    for (let [name, type] of mapping) {
      if (getTypeClass(type) === "TypeVariable") throw new IvyTypeError("no type inferred for " + name) // maybe allow this
      if (constraint.left.name === name && (getTypeClass(type) !== constraint.right)) {
        throw new IvyTypeError("type " + typeToString(type) + " does not match typeclass " + constraint.right)
      }
    }
  }
  return mapping // doesn't do anything to it for now
}

function unify(left: Type, right: Type, constraints: TypeConstraint[], mapping: TypeMap) {
  if (isSameType(left, right)) return applyConstraints(constraints, mapping) // identical; trivially satisfied; move on
  if (isTypeVariable(left)) {
    return substitute(left, right, constraints, mapping)
  }
  if (isTypeVariable(right)) {
    return substitute(right, left, constraints, mapping)
  }
  let typeClass = getTypeClass(left) 
  if (typeClass !== getTypeClass(right)) throw new IvyTypeError("incompatible types: " + typeToString(left) + " and " + typeToString(right))
  switch (typeClass) {
    case "Primitive": throw new IvyTypeError("incompatible types: " + left + " and " + right) // we know they're not the same type from the above check
    case "List": return unify((left as List).elementType, (right as List).elementType, constraints, mapping)
    case "Hash": {
      let leftHashFunction = (left as Hash).hashFunction
      let rightHashFunction = (right as Hash).hashFunction
      if (leftHashFunction !== rightHashFunction) throw new IvyTypeError("incompatible hash functions: " + leftHashFunction + " and " + rightHashFunction)
      return unify((left as Hash).inputType, (right as Hash).inputType, constraints, mapping)
    }
    case "Other":
      throw new IvyTypeError("value of type " + left + " used improperly")
  }
}

function substitute(left: TypeVariable, right: Type, constraints: TypeConstraint[], mapping: TypeMap): TypeMap {
  checkOccurs(left, right) // check if it's an infinitely recurrent type
  if (isList(right)) {
    // type variables can't be Lists, at least for now (lists are only used as literals for checkMultiSig; they can't be compared or hashed)
    throw new IvyTypeError("type variable " + left.name + " can only be bound to a Primitive or Hash type, not a List type")
  }
  if (getTypeClass(right) === "Other") {
    throw new IvyTypeError("type variable " + left.name + " can only be bound to a Primitive or Hash type, not a " + right + " type")
  }

  if (isTypeVariable(right)) {
    mapping.set(right.name, left)
  }

  const replace = (typeVariable) => {
    return isSameType(typeVariable, left) ? right : typeVariable
  }
  const newConstraints = mapOverConstraints(replace, constraints)
  const newMapping = mapOverTypeMap(replace, mapping)
  newMapping.set(left.name, right)

  return applyConstraints(newConstraints, newMapping)
}

export class Scope {
  public variables: Map<string, Type>
  public constraints: TypeConstraint[]
  public typeClassConstraints: TypeClassConstraint[]
  public typeIndex: number
  public clauseName?: string

  constructor() {
    this.variables = new Map<string, Type>()
    this.constraints = []
    this.typeClassConstraints = []
    this.typeIndex = 0
  }

  getVariable(item: Variable): Type {
    let variableType = this.variables.get(scopedName(item)) || this.variables.get(item.identifier)
    if (variableType === undefined) {
      throw new NameError("undefined variable: " + item.identifier)
    } else {
      return variableType
    }
  }

  setVariable(item: Parameter | Variable, type: Type) {
    if (this.variables.has(scopedName(item)) || this.variables.has(item.identifier))
        throw new NameError("variable " + name + " already assigned")
    this.variables.set(scopedName(item), type)
  }

  setVariableWithTypeClass(name: string, typeClass: TypeClass) {
    let newTypeVariable = this.getNewTypeVariable()
    this.addTypeClassConstraint(newTypeVariable, typeClass)
    this.variables.set(name, newTypeVariable)
  }

  getNewTypeVariable(): TypeVariable {
    let newName = "t" + this.typeIndex
    this.typeIndex++
    return { type: "typeVariable", name: newName }
  }

  replaceWithNewTypeVariables(type: Type): Type {
    let typeVariablesMap = new Map<string, Type>()
    addTypeVariablesToMap(type, this, typeVariablesMap)
    return substituteTypeWithMap(type, typeVariablesMap)
  }

  substituteTypeSig(typeSig: TypeSignature) {
    // substitute the type variables in the type signature
    // so they don't conflict with existing ones

    let typeVariablesMap = generateMapFromTypeString(typeSig, this)

    let newInputs = typeSig.inputs.map((input) => (substituteTypeWithMap(input, typeVariablesMap)))
    let newOutput = substituteTypeWithMap(typeSig.output, typeVariablesMap)
    return createTypeSignature(newInputs, newOutput)
  }

  addTypeConstraint(left: Type, right: Type) {
    this.constraints.push({ type: "typeConstraint", left: left, right: right})
  }

  addTypeClassConstraint(left: TypeVariable, right: TypeClass) {
    this.typeClassConstraints.push({ type: "typeClassConstraint", left: left, right: right})
  }
}

function generateMapFromTypeString(typeSig: TypeSignature, scope: Scope) {
  let typeVariablesMap = new Map<string, Type>()

  for (let type of typeSig.inputs) {
    addTypeVariablesToMap(type, scope, typeVariablesMap)
  }
  addTypeVariablesToMap(typeSig.output, scope, typeVariablesMap)

  return typeVariablesMap
}

function addTypeVariablesToMap(type: Type, scope: Scope, typeVariablesMap: Map<string, Type>) {
  return mapOverType((typeVariable) => {
    if (typeVariablesMap.get(typeVariable.name) === undefined) {
      typeVariablesMap.set(typeVariable.name, scope.getNewTypeVariable())
    }
    return typeVariable
  }, type)
}

function substituteTypeWithMap(type: Type, typeVariablesMap: Map<string, Type>): Type {
  return mapOverType((typeVariable) => {
      let newTypeVariable = typeVariablesMap.get(typeVariable.name)
      if (newTypeVariable === undefined) {
        throw new BugError("unexpectedly unknown type variable: " + typeVariable.name)
      } else {
        return newTypeVariable
      }
    }, type)
}

export function matchTypes(firstType: Type, secondType: Type, scope: Scope): Type {
  if (isTypeVariable(firstType) || isTypeVariable(secondType)) {
    // add a type constraint
    scope.addTypeConstraint(firstType, secondType)
    return secondType
  }
  let typeClass = getTypeClass(firstType)
  if (typeClass !== getTypeClass(secondType)) {
    throw new IvyTypeError("got " + typeToString(secondType) + ", expected " + typeToString(firstType))
  }
  switch (typeClass) {
    case "Primitive":
    case "Other":
      if (firstType !== secondType) throw new IvyTypeError("got " + typeToString(secondType) + 
                                                           ", expected " + typeToString(firstType))
      return firstType
    case "List":
      return { type: "listType", elementType: matchTypes((firstType as List).elementType, 
                                                         (secondType as List).elementType,
                                                         scope) }
    case "Hash":
      if (!isHash(firstType) || !isHash(secondType)) throw new BugError("type guard surprisingly failed")
      if (firstType.hashFunction !== secondType.hashFunction) throw new IvyTypeError("cannot unify " + typeToString(firstType) + " with " +
                                                                                     typeToString(firstType))
      return { type: "hashType", hashFunction: firstType.hashFunction, inputType: matchTypes(firstType.inputType, secondType.inputType, scope) }
    default:
      throw new BugError("type class should have been handled")
  }
}


export function unifyFunction(typeSignature: TypeSignature, inputTypes: Type[], scope: Scope): Type {
  // typecheck some inputs against the function's type signature
  // also maybe compute the output type and/or more specific input types

  let typeSigInputs = typeSignature.inputs
  if (inputTypes.length !== typeSignature.inputs.length) {
    throw new IvyTypeError("expected " + inputTypesToString(typeSigInputs) + ", got " + inputTypesToString(inputTypes))
  }
  for (let i = 0; i < inputTypes.length; i++) {
    const firstType = typeSigInputs[i]
    const secondType = inputTypes[i]
    matchTypes(firstType, secondType, scope)
  }
  return typeSignature.output
}

export function typeCheckExpression(expression: Expression, scope: Scope): Type {
  switch (expression.type) {
    case "instructionExpression":
      let typeSig = scope.substituteTypeSig(getTypeSignature(expression.instruction))
      let inputTypes = expression.args.map((arg) => typeCheckExpression(arg, scope))
      return unifyFunction(typeSig, inputTypes, scope)
    case "literal":
      return expression.literalType
    case "variable":
      return scope.getVariable(expression)
    case "listLiteral":
      if (expression.values.length === 0) {
        throw new IvyTypeError("lists cannot be empty")
      }
      let unifiedType = expression.values.map((exp) => typeCheckExpression(exp, scope))
                                         .reduce((firstType, secondType) => matchTypes(firstType, secondType, scope))
      return { type: "listType", elementType: unifiedType }
    case "contractExpression": {
      let programType = typeCheckExpression(expression.program, scope)
      let valueType = typeCheckExpression(expression.value, scope)
      if (programType !== "Program") {
        throw new IvyTypeError("expected " + expression.program.identifier + "to have type Program, got " + typeToString(programType)) 
      }
      if (valueType !== "Value") {
        throw new IvyTypeError("expected" + expression.value.identifier + " to have type Value, got " + typeToString(valueType))
      }
      return "Contract"
    }
    case "storedValue": {
      return "Value"
    }
  }
}

export function typeCheckAssertion(assertion: Assertion, scope: Scope) {
  let expressionType = typeCheckExpression(assertion.expression, scope)
  if (expressionType !== "Boolean") {
    throw new IvyTypeError("verify statement expects a Boolean, got " + typeToString(expressionType))
  }
}

export function typeCheckReturn(returnStatement: Return, scope: Scope) {
  let valueType = typeCheckExpression(returnStatement.value, scope)
  if (valueType !== "Value") {
    throw new IvyTypeError("return statement expects a Value, got " + typeToString(valueType))
  }
}

export function typeCheckOutput(output: Output, scope: Scope) {
  let contractType = typeCheckExpression(output.contract, scope)
  if (contractType !== "Contract") {
    throw new IvyTypeError("return statement expects a Contract, got " + typeToString(contractType))
  }
}

export function typeCheckStatement(statement: Statement, scope: Scope) {
  switch (statement.type) {
    case "assertion": return typeCheckAssertion(statement, scope)
    case "returnStatement": return typeCheckReturn(statement, scope)
    case "output": return typeCheckOutput(statement, scope)
  }
}

export function typeCheckClause(clause: Clause, scope: Scope) {
  checkClauseValueParameters(clause)
  scope.clauseName = clause.name
  for (let parameter of clause.parameters) {
    if (parameter.itemType === undefined) throw new BugError("parameter type unexpectedly undefined")
    if (isTypeClass(parameter.itemType)) {
      scope.setVariableWithTypeClass(parameter.identifier, parameter.itemType)
    } else {
      scope.setVariable(parameter, parameter.itemType)
    }
  }
  for (let assertion of clause.assertions) {
    typeCheckAssertion(assertion, scope)
  }
  for (let output of clause.outputs) {
    typeCheckOutput(output, scope)
  }
  if (clause.returnStatement) {
    typeCheckReturn(clause.returnStatement, scope)
  }
}

function checkUniqueClauseNames(clauses: Clause[]) {
  let clauseNames = clauses.map((clause) => clause.name)
  if (new Set(clauseNames).size !== clauseNames.length) {
    throw new NameError("clause names must be unique")
  }
}

function checkMultiSigArgumentCounts(contract: RawContract) {
  mapOverAST((node: ASTNode) => {
    switch (node.type) {
      case "instructionExpression": {
        // check checkMultiSig argument counts
        if (node.instruction == "checkTxMultiSig") {
          let pubKeys = node.args[0] as ListLiteral
          let sigs = node.args[1] as ListLiteral
          if (sigs.values.length > pubKeys.values.length) {
            throw new IvyTypeError("number of public keys passed to checkMultiSig must be greater than or equal to number of signatures")
          }
        }
        return node
      }
      default: return node
    }
  }, contract)
}

export function checkContractValueParameter(contract: RawContract): RawContract {
  if (contract.parameters.length === 0) {
    throw new IvyTypeError("contract must have a Value parameter")
  }
  let valueParameter = contract.parameters[contract.parameters.length - 1]
  if (valueParameter.itemType !== "Value") {
    throw new IvyTypeError("last contract parameter must have type Value")
  }
  for (let i = 0; i < contract.parameters.length - 1; i++) {
    if (contract.parameters[i].itemType === "Value") {
      throw new IvyTypeError("contract can only have one parameter of type Value")
    }
  }
  let newContract = mapOverAST((node: ASTNode) => {
    switch(node.type) {
      case "contractExpression": 
      case "returnStatement": {
        if (node.value.identifier === valueParameter.identifier) {
          return {
            ...node,
            value: {
              location: node.value.location,
              identifier: node.value.identifier,
              type: "storedValue",
            }
          }
        }
        return node
      }
      default: return node
    }
  }, contract) as RawContract
  for (let clause of newContract.clauses) {
    // Check that the value is properly disposed of (via exactly one
    // output or return) in each clause.
    checkValueUses(valueParameter, clause)
  }
  return newContract
}

// Ensure that for each value param in the clause, it's disposed of
// exactly once (also in the clause).
function checkClauseValueParameters(clause: Clause) {
  for (let parameter of clause.parameters) {
    if (parameter.itemType == "Value") {
      checkValueUses(parameter, clause)
    }
  }
}

// Check that the given value parameter is disposed of exactly once in
// the given clause.
function checkValueUses(valueParameter: Parameter, clause: Clause) {
  let found = 0
  for (let output of clause.outputs) {
    if (output.contract.value.identifier == valueParameter.identifier) {
      found++
    }
  }
  if (clause.returnStatement !== undefined) {
    if (clause.returnStatement.value.identifier == valueParameter.identifier) {
      found++
    }
  }
  if (found == 0) {
    throw new IvyTypeError("value parameter " + valueParameter.identifier + " unused in clause " + clause.name)
  }
  if (found > 1) {
    throw new IvyTypeError("value parameter " + valueParameter.identifier + " used more than once in clause " + clause.name)
  }
}

export function typeCheckContract(initialContract: RawContract): RawContract {
  let contract = checkContractValueParameter(initialContract)
  checkUniqueClauseNames(contract.clauses)

  let contractScope = new Scope()
  for (let parameter of contract.parameters) {
    if (parameter.itemType === undefined) throw new BugError("parameter type unexpectedly undefined")
    if (parameter.itemType === "Hash") {
      contractScope.setVariableWithTypeClass(parameter.identifier, "Hash")
    } else if (parameter.itemType === "Signature") {
      throw new IvyTypeError("Signatures cannot be used as contract arguments")
    } else {
      contractScope.setVariable(parameter, parameter.itemType)
    }
  }
  for (let clause of contract.clauses) {
    typeCheckClause(clause, contractScope)
  }
  let constrainedTypes = applyConstraints([...contractScope.constraints])
  let inferredTypes = applyTypeClassConstraints(contractScope.typeClassConstraints, constrainedTypes)
  contractScope.variables = mapOverTypeMap((typeVariable) => {
    let replacedType = inferredTypes.get(typeVariable.name)
    if (replacedType === undefined) return typeVariable
    return replacedType
  }, contractScope.variables)

  checkMultiSigArgumentCounts(contract)

  // map inferred types onto AST
  return mapOverAST((node: ASTNode) => {
    switch (node.type) {
      case "parameter": {
        let inferredType = contractScope.variables.get(scopedName(node))
        if (inferredType === undefined) throw new BugError("no type found for " + node.identifier)
        if (isTypeVariable(inferredType)) throw new BugError("type surprisingly uninferred for " + node.identifier)
        if (isList(inferredType)) throw new BugError("parameters cannot have type List")
        if (inferredType == "Contract" || inferredType == "SigHash") throw new BugError("parameters cannot have type " + inferredType)
        let newParameter: Parameter = {
          ...node,
          itemType: inferredType
        }
        return newParameter
      }
      default: return node
    }
  }, contract) as RawContract

}

