import {
  FinalOperation
} from '../intermediate'

import {
  getOpcodes
} from './instructions'

export default function toOpcodes(ops: FinalOperation[]): string[] {
  let newOps: string[] = []

  let emit = (o: string) => {
    newOps.push(o)
  }

  for (let op of ops) {
    switch (op.type) {
      case "pick": {
        emit(op.depth.toString())
        emit("PICK")
        break
      }
      case "roll": {
        emit(op.depth.toString())
        emit("ROLL")
        break
      }
      case "instructionOp": {
        let instructionOpcodes = getOpcodes(op.expression.instruction)
        instructionOpcodes.map(emit)
        break
      }
      case "op": {
        emit(op.name)
        break
      }
      case "push": {
        if (op.literalType === "Boolean") {
          emit(op.value === "true" ? "TRUE" : "FALSE")
        } else {
          emit(op.value)
        }
        break
      }

      case "beginIf": {
        if (op.elseTag === undefined) {
          // if (cond) { ifClause } -> cond NOT JUMPIF:$x ifClause $x
          emit("NOT")
          emit("JUMPIF:$" + op.endTag)
        } else {
          // if (cond) { ifClause } else { elseClause } -> cond NOT JUMPIF:$x ifClause JUMP:$y $x elseClause $y
          // TODO(bobg): this would be slightly better as:
          //   cond JUMPIF:$x elseClause JUMP:$y $x ifClause $y
          // (reversing the ifClause and elseClause, and removing the NOT)
          // but that will require a structural change to the compiler.
          emit("NOT")
          emit("JUMPIF:$" + op.elseTag)
        }
        break
      }
      case "else": {
        emit("JUMP:$" + op.endTag)
        emit("$" + op.elseTag)
        break
      }
      case "endIf": {
        emit("$" + op.endTag)
        break
      }
      case "pushParameter": {
        emit("<" + op.identifier + ">")
        break
      }
      case "drop": {
        emit("DROP")
        break
      }
    }
  }

  return newOps
}
