import * as crypto from 'crypto'

export function sha256(buf: Buffer): Buffer {
  return crypto.createHash('sha256').update(buf).digest()
}