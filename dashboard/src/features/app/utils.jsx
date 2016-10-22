import React from 'react'

export const navIcon = (name, styles) => {
  let active = false
  const icon = require(`assets/images/navigation/${name}.png`)

  try {
    active = require(`assets/images/navigation/${name}-active.png`)
  } catch (err) { /* do nothing */ }
  return (
    <span className={styles.iconWrapper}>
      <img className={styles.icon} src={icon}/>
      {active && <img className={styles.activeIcon} src={active}/>}
    </span>
  )
}
