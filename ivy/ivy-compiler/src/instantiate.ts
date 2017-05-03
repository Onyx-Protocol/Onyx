import {
  Template, countTemplateParams
} from './template'

import {
  AssemblerError, isAssemblerError, assemble
} from './cvm/assemble'

export default function instantiate(template: Template, args: (Buffer|number)[]): Buffer {
  const numArgs = countTemplateParams(template)
  if (numArgs !== args.length) throw "expected " + numArgs + " arguments, got " + args.length
  let instructions = template.instructions
  let body = instructions.slice(numArgs)
  let argOps = [...args].reverse()
  let opcodes = ([] as any[]).concat(argOps, body)
  let res = assemble(opcodes)
  if (isAssemblerError(res)) {
    throw res
  }
  return res
}
