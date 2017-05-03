import {
  mapOverAST,
  RawContract,
  ASTNode
} from './ast'

import {
  NameError
} from './errors'

export function referenceCheck(contract: RawContract): RawContract {
  // annotate parameters and variables with their scope and reference count within that clause
  // there are two mappings over ASTs—one that finds the clauses, and one that maps over those clauses
  // also throw an error on undefined or unused variables

  let contractCounts = new Map<string, number>()
  let assetAmountParams = new Set<string>()

  for (let parameter of contract.parameters) {
    if (parameter.itemType === "AssetAmount") {
      contractCounts.set(parameter.identifier + ".asset", 0)
      contractCounts.set(parameter.identifier + ".amount", 0)
      assetAmountParams.add(parameter.identifier)
    } else {
      contractCounts.set(parameter.identifier, 0)
    }
  }

  let result = mapOverAST((node: ASTNode) => {
    switch (node.type) {
      case "clause": {
        let clauseName = node.name

        let clauseParameters = node.parameters.map(param => { return {
          ...param,
          scope: clauseName
        }})

        let counts = new Map<string, number>()

        for (let parameter of contract.parameters) {
          if (assetAmountParams.has(parameter.identifier)) {
            counts.set(parameter.identifier + ".asset", 0)
            counts.set(parameter.identifier + ".amount", 0)
          } else {
            counts.set(parameter.identifier, 0)
          }
        }

        for (let parameter of clauseParameters) {
          if (contractCounts.has(parameter.identifier)) 
            throw new NameError("parameter " + parameter.identifier + " is already defined")
          counts.set(parameter.identifier, 0)
        }

        let mappedClause = mapOverAST((node: ASTNode) => {
            switch (node.type) {
              case "variable": {
                let identifier = node.identifier
                if (identifier.endsWith(".assetAmount")) {
                  // TODO(bobg): undo this hack
                  identifier = identifier.substring(0, identifier.length - 12)
                }
                let currentContractCount: number|undefined
                let currentCount: number|undefined
                if (assetAmountParams.has(identifier)) {
                  currentContractCount = contractCounts.get(identifier + ".asset") // could also be +".amount", they're the same number
                  currentCount = counts.get(identifier + ".asset")
                } else {
                  currentContractCount = contractCounts.get(identifier)
                  currentCount = counts.get(identifier)
                }
                if (currentCount === undefined)
                  throw new NameError("unknown variable: " + identifier)
                if (assetAmountParams.has(identifier)) {
                  counts.set(identifier + ".asset", currentCount + 1)
                  counts.set(identifier + ".amount", currentCount + 1)
                } else {
                  counts.set(identifier, currentCount + 1)
                }
                if (currentContractCount !== undefined) {
                  if (assetAmountParams.has(identifier)) {
                    contractCounts.set(identifier + ".asset", currentContractCount + 1)
                    contractCounts.set(identifier + ".amount", currentContractCount + 1)
                  } else {
                    contractCounts.set(identifier, currentContractCount + 1)
                  }
                  return node
                } else {
                  return {
                    ...node,
                    scope: clauseName
                  }
                }
              }
              default: return node
            }
          }, node)
        for (let parameter of clauseParameters) {
          if (counts.get(parameter.identifier) === 0) {
            throw new NameError("unused variable in clause " + clauseName + ": " + parameter.identifier)
          }
        }
        return {
          ...mappedClause,
          referenceCounts: counts,
          parameters: clauseParameters
        }
      }
      default: return node
    }
  }, contract)
  for (let [key, value] of contractCounts) {
    if (value === 0) {
      throw new NameError("unused parameter: " + key)
    }
  }
  return {
    ...result,
    referenceCounts: contractCounts
  } as RawContract
}
