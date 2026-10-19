import React from 'react'

const API = import.meta.env.VITE_API_URL || 'http://localhost:8080'

export default function CodeEntry({ code, onCodeChange, taxi, onTaxi, error, onError }) {
  React.useEffect(() => {
    const trimmed = code.trim().toUpperCase()
    if (trimmed.length !== 4) {
      onTaxi(null)
      onError('')
      return
    }
    let cancelled = false
    ;(async () => {
      try {
        const res = await fetch(`${API}/api/v1/taxi/${trimmed}`)
        const data = await res.json()
        if (cancelled) return
        if (!res.ok) {
          onTaxi(null)
          onError(data.error || 'Taxi not found')
          return
        }
        onTaxi(data)
        onError('')
      } catch {
        if (!cancelled) {
          onTaxi(null)
          onError('Cannot reach GuzoPay')
        }
      }
    })()
    return () => {
      cancelled = true
    }
  }, [code, onTaxi, onError])

  return (
    <div className="space-y-3">
      <label className="block text-sm font-medium text-mpesa-ink/80">Enter vehicle code</label>
      <input
        value={code}
        onChange={(e) => onCodeChange(e.target.value.replace(/[^a-zA-Z0-9]/g, '').slice(0, 4).toUpperCase())}
        placeholder="4A2Q"
        maxLength={4}
        inputMode="text"
        autoComplete="off"
        className="w-full rounded-xl border-2 border-mpesa-green/30 bg-white px-4 py-3 text-center text-2xl font-bold tracking-[0.35em] text-mpesa-ink outline-none focus:border-mpesa-green"
      />
      {error && <p className="text-sm text-red-600">{error}</p>}
      {taxi && (
        <div className="rounded-xl bg-white px-4 py-3 shadow-sm ring-1 ring-mpesa-green/15">
          <p className="text-xs font-semibold uppercase tracking-wide text-mpesa-green">Driver</p>
          <p className="text-lg font-bold text-mpesa-ink">{taxi.name}</p>
          <p className="text-sm text-mpesa-ink/70">{taxi.route}</p>
        </div>
      )}
    </div>
  )
}
