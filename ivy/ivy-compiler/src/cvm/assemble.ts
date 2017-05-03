export type AssemblerToken = string | Buffer | number

export type AssemblerError = {
  type: "assemblerError",
  token: AssemblerToken,
  message: string
}

export function isAssemblerError(x: any): x is AssemblerError {
  return x.type == "assemblerError"
}

let table = {
  "FALSE": 0x00,
  "TRUE":  0x51,

  "1NEGATE": 0x4f,
  "NOP":     0x61,

  "VERIFY":         0x69,
  "FAIL":           0x6a,
  "CHECKPREDICATE": 0xc0,

  "TOALTSTACK":   0x6b,
  "FROMALTSTACK": 0x6c,
  "2DROP":        0x6d,
  "2DUP":         0x6e,
  "3DUP":         0x6f,
  "2OVER":        0x70,
  "2ROT":         0x71,
  "2SWAP":        0x72,
  "IFDUP":        0x73,
  "DEPTH":        0x74,
  "DROP":         0x75,
  "DUP":          0x76,
  "NIP":          0x77,
  "OVER":         0x78,
  "PICK":         0x79,
  "ROLL":         0x7a,
  "ROT":          0x7b,
  "SWAP":         0x7c,
  "TUCK":         0x7d,

  "CAT":         0x7e,
  "SUBSTR":      0x7f,
  "LEFT":        0x80,
  "RIGHT":       0x81,
  "SIZE":        0x82,
  "CATPUSHDATA": 0x89,

  "INVERT":      0x83,
  "AND":         0x84,
  "OR":          0x85,
  "XOR":         0x86,
  "EQUAL":       0x87,
  "EQUALVERIFY": 0x88,

  "1ADD":      0x8b,
  "1SUB":      0x8c,
  "2MUL":      0x8d,
  "2DIV":      0x8e,
  "NEGATE":    0x8f,
  "ABS":       0x90,
  "NOT":       0x91,
  "0NOTEQUAL": 0x92,
  "ADD":       0x93,
  "SUB":       0x94,
  "MUL":       0x95,
  "DIV":       0x96,
  "MOD":       0x97,
  "LSHIFT":    0x98,
  "RSHIFT":    0x99,
  "BOOLAND":   0x9a,
  "BOOLOR":             0x9b,
  "NUMEQUAL":           0x9c,
  "NUMEQUALVERIFY":     0x9d,
  "NUMNOTEQUAL":        0x9e,
  "LESSTHAN":           0x9f,
  "GREATERTHAN":        0xa0,
  "LESSTHANOREQUAL":    0xa1,
  "GREATERTHANOREQUAL": 0xa2,
  "MIN":                0xa3,
  "MAX":                0xa4,
  "WITHIN":             0xa5,

  "SHA256":        0xa8,
  "SHA3":          0xaa,
  "CHECKSIG":      0xac,
  "CHECKMULTISIG": 0xad,
  "TXSIGHASH":     0xae,
  "BLOCKHASH":     0xaf,

  "CHECKOUTPUT": 0xc1,
  "ASSET":       0xc2,
  "AMOUNT":      0xc3,
  "PROGRAM":     0xc4,
  "MINTIME":     0xc5,
  "MAXTIME":     0xc6,
  "TXDATA":      0xc7,
  "ENTRYDATA":   0xc8,
  "INDEX":       0xc9,
  "ENTRYID":     0xca,
  "OUTPUTID":    0xcb,
  "NONCE":       0xcc,
  "NEXTPROGRAM": 0xcd,
  "BLOCKTIME":   0xce
}

export function assemble(tokens: AssemblerToken[]): Buffer|AssemblerError {
  let res: number[] = []
  let locations = {}
  let unresolved = {}
  let handleJump = function(label: string, bytecode: number) {
    res.push(bytecode)
    let l = res.length
    res.push(0, 0, 0, 0)
    if (unresolved[label] === undefined) {
      unresolved[label] = []
    }
    unresolved[label].push(l)
  }
  for (let token of tokens) {
    if (typeof token === "string") {
      let code = table[token]
      if (code !== undefined) {
        res.push(code)
        continue
      }
      if (token.startsWith("JUMP:")) {
        handleJump(token.substring(5), 0x63)
        continue
      }
      if (token.startsWith("JUMPIF:")) {
        handleJump(token.substring(7), 0x64)
        continue
      }
      if (token.startsWith("$")) {
        let seen = locations[token]
        if (seen !== undefined) {
          return {
            type: "assemblerError",
            token: token,
            message: "jump target redefined"
          }
        }
        locations[token] = res.length
        continue
      }
      if (token.startsWith("0x")) {
        if (token.length % 2 != 0) {
          return {
            type: "assemblerError",
            token: token,
            message: "odd number of literal hex characters"
          }
        }
        for (let i = 2; i < token.length; i += 2) {
          let byte = Number.parseInt(token.substr(i, 2), 16)
          if (byte !== byte) { // NaN
            return {
              type: "assemblerError",
              token: token,
              message: "illegal character in hex literal"
            }
          }
          res.push(byte)
        }
        continue
      }
      // TODO(bobg): add string literals
      let num = Number.parseInt(token)
      if (num === num) { // NaN
        pushdata(res, uint64Bytes(num))
        continue
      }
      // xxx skip unrecognized tokens for now
      continue
    }
    if (typeof token === "number") {
      pushdata(res, uint64Bytes(token))
      continue
    }
    pushdata(res, token)
  }
  for (let label in unresolved) {
    let location = locations[label]
    if (location === undefined) {
      return {
        type: "assemblerError",
        token: label,
        message: "undefined jump target"
      }
    }

    let uses = unresolved[label]
    for (let use of uses) {
      // Fill res[use .. use+3] with the uint32le representation of
      // location.
      let rep = uint64le(location)
      res[use] = rep[0]
      res[use+1] = rep[1]
      res[use+2] = rep[2]
      res[use+3] = rep[3]
    }
  }
  return Buffer.from(res)
}

function uint64le(num: number): Uint8Array {
  let res = new Uint8Array(8)
  for (let i = 0; i < 8; i++) {
    res[i] = num & 255
    num >>= 8
  }
  return res
}

function uint64Bytes(num: number): number[] {
  let arr = uint64le(num)
  let len = 8
  for (let i = 7; i >= 0; i--) {
    if (arr[i] != 0) {
      break
    }
    len--
  }
  let res: number[] = []
  for (let i = 0; i < len; i++) {
    res.push(arr[i])
  }
  return res
}

function pushdata(res: number[], buf: Buffer|number[]) {
  if (buf.length == 0) {
    res.push(0)
    return
  }
  if (buf.length <= 75) {
    res.push(buf.length) // OP_DATA_<len>
  } else if (buf.length < 1<<8) {
    res.push(0x4c) // OP_PUSHDATA1
    res.push(buf.length)
  } else {
    let lrep = uint64le(buf.length)
    if (buf.length < 1<<16) {
      res.push(0x4d) // OP_PUSHDATA2
      res.push(lrep[0], lrep[1])
    } else {
      res.push(0x4e) // OP_PUSHDATA4
      res.push(lrep[0], lrep[1], lrep[2], lrep[3])
    }
  }
  for (let b of buf) {
    res.push(b)
  }
}
