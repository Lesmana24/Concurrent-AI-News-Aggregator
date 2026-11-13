import { useState, useEffect } from 'react'

export interface WibRemaining {
  hours: number
  minutes: number
  seconds: number
  formatted: string
  totalSeconds: number
}

export function getWibMidnightRemaining(): WibRemaining {
  const now = new Date()
  const utcMs = now.getTime() + now.getTimezoneOffset() * 60000
  const wibMs = utcMs + 7 * 3600000
  const wibDate = new Date(wibMs)

  const nextMidnightWib = new Date(wibMs)
  nextMidnightWib.setHours(24, 0, 0, 0)

  const diffMs = Math.max(0, nextMidnightWib.getTime() - wibDate.getTime())
  const totalSeconds = Math.floor(diffMs / 1000)
  const hours = Math.floor(totalSeconds / 3600)
  const minutes = Math.floor((totalSeconds % 3600) / 60)
  const seconds = totalSeconds % 60

  return {
    hours,
    minutes,
    seconds,
    formatted: `${hours}j ${minutes}m ${seconds}d`,
    totalSeconds,
  }
}

export function useWibCountdown(onMidnight?: () => void): string {
  const [countdown, setCountdown] = useState<string>(() => getWibMidnightRemaining().formatted)

  useEffect(() => {
    let triggeredMidnight = false

    const tick = () => {
      const remaining = getWibMidnightRemaining()
      setCountdown(remaining.formatted)

      if (remaining.totalSeconds === 0 && !triggeredMidnight) {
        triggeredMidnight = true
        onMidnight?.()
      } else if (remaining.totalSeconds > 0) {
        triggeredMidnight = false
      }
    }

    tick()
    const interval = setInterval(tick, 1000)
    return () => clearInterval(interval)
  }, [onMidnight])

  return countdown
}

