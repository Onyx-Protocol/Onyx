/**
 * Calculates the average change per second of a variable sampled at various times.
 */
export class DeltaSampler {
  constructor({sampleTtl = 60*1000, maxSamples = 30} = {}) {
    this.sampleTtl = sampleTtl
    this.maxSamples = maxSamples
    this.samples = []
  }

  sample(value) {
    this.samples.push({
      value,
      time: Date.now(),
    })

    if (this.samples.length > this.maxSamples) {
      this.samples.shift()
    }

    return this.average()
  }

  /**
   * Returns the average growth of the value per second.
   * Algorithm: sum the changes
   */
  average() {
    const cutoff = Date.now() - this.sampleTtl

    let earliest = null
    let latest = null

    for (let i = 0; i < this.samples.length; i++) {
      const s = this.samples[i]
      if (s.time < cutoff) continue
      if (earliest === null) earliest = s
      latest = s
    }

    if (earliest === latest) {
      return NaN
    }

    return 1000 * (latest.value - earliest.value) / (latest.time - earliest.time)
  }
}

export const humanizeDuration = seconds => {
  if (seconds == 0) {
    return '0s'
  }

  const sec = 1
  const min = 60 * sec
  const hr = 60 * min
  const day = 24 * hr

  let bigUnit, littleUnit, bigLabel, littleLabel

  if (seconds >= day) {
    bigUnit = day
    littleUnit = hr
    bigLabel = 'd'
    littleLabel = 'h'
  } else if (seconds >= hr) {
    bigUnit = hr
    littleUnit = min
    bigLabel = 'h'
    littleLabel = 'm'
  } else {
    bigUnit = min
    littleUnit = sec
    bigLabel = 'm'
    littleLabel = 's'
  }

  const bigVal = Math.floor(seconds / bigUnit)
  const littleVal = Math.round((seconds % bigUnit) / littleUnit)

  // Rounding may give us little-unit vals of 60s, 24h, etc.
  if (littleVal == bigUnit / littleUnit) {
    return `${bigVal + 1}${bigLabel}`
  }

  const big = `${bigVal}${bigLabel}`
  const little = `${littleVal}${littleLabel}`

  // Don't show little unit if the big unit is in double digits
  if (bigVal > 9 || littleVal == 0) {
    return big
  }

  // For values that round to under 60 seconds
  if (bigVal == 0) {
    return little
  }

  return `${big} ${little}`
}
