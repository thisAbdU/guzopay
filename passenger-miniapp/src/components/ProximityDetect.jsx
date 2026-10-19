import React from 'react'

const API = import.meta.env.VITE_API_URL || 'http://localhost:8080'

export default function ProximityDetect({ onSelect, pollMs = 3000 }) {
  const [taxi, setTaxi] = React.useState(null)
  const [scanning, setScanning] = React.useState(true)

  React.useEffect(() => {
    let alive = true
    const scan = async () => {
      try {
        const res = await fetch(`${API}/api/v1/taxi/nearby`)
        const data = await res.json()
        if (!alive) return
        setTaxi(data.taxis?.[0] ?? null)
        setScanning(false)
      } catch {
        if (alive) setScanning(false)
      }
    }
    scan()
    const id = setInterval(scan, pollMs)
    return () => {
      alive = false
      clearInterval(id)
    }
  }, [pollMs])

  if (scanning && !taxi) {
    return (
      <div className="rounded-2xl border border-mpesa-green/20 bg-white/80 px-4 py-3 text-sm text-mpesa-ink/70">
        Scanning for nearby taxis…
      </div>
    )
  }

  if (!taxi) return null

  return (
    <button
      type="button"
      onClick={() => onSelect(taxi.vehicle_code)}
      className="w-full rounded-2xl border-2 border-mpesa-green bg-mpesa-light px-4 py-4 text-left shadow-sm transition active:scale-[0.99]"
    >
      <p className="text-xs font-semibold uppercase tracking-wide text-mpesa-green">Nearby taxi found</p>
      <p className="mt-1 text-lg font-bold text-mpesa-ink">Taxi #{taxi.vehicle_code}</p>
      <p className="text-sm text-mpesa-ink/70">
        {taxi.name} · {taxi.route}
      </p>
      <p className="mt-2 text-sm font-semibold text-mpesa-green">Tap to continue →</p>
    </button>
  )
}
