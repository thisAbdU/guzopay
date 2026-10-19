import React from 'react'
import { Radio, RadioTower } from 'lucide-react'
import AnalyticsHeader from './components/AnalyticsHeader.jsx'
import AudioAlert, { playDing } from './components/AudioAlert.jsx'
import FareTiersEditor from './components/FareTiersEditor.jsx'
import RecentTransactionsTable from './components/RecentTransactionsTable.jsx'
import WithdrawPanel from './components/WithdrawPanel.jsx'

const API = import.meta.env.VITE_API_URL || 'http://localhost:8080'
const WS = import.meta.env.VITE_WS_URL || 'ws://localhost:8080'

export default function App() {
  const [drivers, setDrivers] = React.useState([])
  const [vehicle, setVehicle] = React.useState(() => localStorage.getItem('guzopay_vehicle') || '')
  const [dash, setDash] = React.useState(null)
  const [txs, setTxs] = React.useState([])
  const [flashIds, setFlashIds] = React.useState(() => new Set())
  const [audioOn, setAudioOn] = React.useState(false)
  const [wsState, setWsState] = React.useState('offline')
  const audioRef = React.useRef(null)
  const audioOnRef = React.useRef(false)

  React.useEffect(() => {
    audioOnRef.current = audioOn
  }, [audioOn])

  React.useEffect(() => {
    fetch(`${API}/api/v1/drivers`)
      .then((r) => r.json())
      .then((data) => {
        const list = data.drivers || []
        setDrivers(list)
        if (!vehicle && list.length === 1) {
          setVehicle(list[0].vehicle_code)
        }
      })
      .catch(console.error)
  }, [])

  React.useEffect(() => {
    if (vehicle) localStorage.setItem('guzopay_vehicle', vehicle)
  }, [vehicle])

  const loadDashboard = React.useCallback(async () => {
    if (!vehicle) return
    const res = await fetch(`${API}/api/v1/driver/${vehicle}/dashboard`)
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || 'Failed to load')
    setDash(data)
    setTxs(data.transactions || [])
  }, [vehicle])

  React.useEffect(() => {
    loadDashboard().catch(console.error)
  }, [loadDashboard])

  React.useEffect(() => {
    if (!vehicle) return undefined
    let ws
    let closed = false
    let retry

    const connect = () => {
      setWsState('connecting')
      ws = new WebSocket(`${WS}/ws/driver/${vehicle}`)
      ws.onopen = () => setWsState('live')
      ws.onclose = () => {
        setWsState('offline')
        if (!closed) retry = setTimeout(connect, 2000)
      }
      ws.onerror = () => ws.close()
      ws.onmessage = (ev) => {
        try {
          const msg = JSON.parse(ev.data)
          if (msg.type === 'payment' && msg.payment) {
            const payment = msg.payment
            setTxs((prev) => [payment, ...prev.filter((t) => t.id !== payment.id)].slice(0, 20))
            setDash((d) =>
              d
                ? {
                    ...d,
                    wallet_balance: msg.wallet_balance ?? d.wallet_balance + payment.amount,
                    today_total: (d.today_total || 0) + payment.amount,
                    trip_count: (d.trip_count || 0) + 1,
                  }
                : d,
            )
            setFlashIds((prev) => new Set(prev).add(payment.id))
            setTimeout(() => {
              setFlashIds((prev) => {
                const next = new Set(prev)
                next.delete(payment.id)
                return next
              })
            }, 1500)
            if (audioOnRef.current) playDing(audioRef)
          }
          if (msg.type === 'payout' || msg.type === 'payout_failed') {
            setDash((d) => (d ? { ...d, wallet_balance: msg.wallet_balance ?? d.wallet_balance } : d))
          }
        } catch (e) {
          console.error(e)
        }
      }
    }

    connect()
    return () => {
      closed = true
      clearTimeout(retry)
      ws?.close()
    }
  }, [vehicle])

  const toggleProximity = async () => {
    if (!dash || !vehicle) return
    const enabled = !dash.proximity_enabled
    const res = await fetch(`${API}/api/v1/driver/${vehicle}/proximity`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ enabled }),
    })
    const data = await res.json()
    if (res.ok) setDash((d) => ({ ...d, proximity_enabled: data.proximity_enabled }))
  }

  if (!vehicle) {
    return (
      <div className="mx-auto min-h-dvh max-w-lg px-4 py-10">
        <p className="text-xs font-bold uppercase tracking-[0.18em] text-cab-green">GuzoPay Driver</p>
        <h1 className="mt-2 text-2xl font-bold text-cab-ink">Select your taxi</h1>
        <p className="mt-1 text-sm text-cab-ink/60">Drivers are loaded from the database seed.</p>
        <ul className="mt-6 space-y-2">
          {drivers.map((d) => (
            <button
              key={d.vehicle_code}
              type="button"
              onClick={() => setVehicle(d.vehicle_code)}
              className="flex w-full flex-col rounded-2xl bg-white px-4 py-3 text-left shadow-sm ring-1 ring-black/5"
            >
              <span className="font-mono text-lg font-bold text-cab-ink">#{d.vehicle_code}</span>
              <span className="text-sm font-medium text-cab-ink">{d.name}</span>
              <span className="text-xs text-cab-ink/55">{d.route}</span>
            </button>
          ))}
          {!drivers.length && (
            <p className="text-sm text-cab-ink/50">No drivers — run seed / start API.</p>
          )}
        </ul>
      </div>
    )
  }

  return (
    <div className="mx-auto min-h-dvh max-w-5xl px-4 py-5 sm:px-6">
      <header className="mb-5 flex flex-wrap items-start justify-between gap-3">
        <div>
          <p className="text-xs font-bold uppercase tracking-[0.18em] text-cab-green">GuzoPay Driver</p>
          <h1 className="text-2xl font-bold text-cab-ink sm:text-3xl">
            Taxi #{vehicle}
            {dash?.driver_name ? ` · ${dash.driver_name}` : ''}
          </h1>
          <p className="mt-1 text-sm text-cab-ink/60">{dash?.route || 'Loading route…'}</p>
          <button
            type="button"
            className="mt-2 text-xs font-semibold text-cab-green underline"
            onClick={() => {
              setVehicle('')
              setDash(null)
              setTxs([])
            }}
          >
            Switch vehicle
          </button>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <span
            className={`inline-flex items-center gap-1.5 rounded-lg px-2.5 py-1.5 text-xs font-bold uppercase ${
              wsState === 'live' ? 'bg-cab-lime/15 text-cab-green' : 'bg-cab-warn/15 text-cab-warn'
            }`}
          >
            <span className={`h-2 w-2 rounded-full ${wsState === 'live' ? 'bg-cab-lime' : 'bg-cab-warn'}`} />
            {wsState}
          </span>
          <AudioAlert
            enabled={audioOn}
            audioRef={audioRef}
            onToggle={() => {
              const next = !audioOn
              setAudioOn(next)
              if (next) playDing(audioRef)
            }}
          />
          <button
            type="button"
            onClick={toggleProximity}
            className={`inline-flex items-center gap-2 rounded-lg border px-3 py-2 text-sm font-semibold ${
              dash?.proximity_enabled
                ? 'border-cab-green bg-cab-green text-white'
                : 'border-cab-ink/20 bg-white text-cab-ink'
            }`}
          >
            {dash?.proximity_enabled ? <RadioTower size={16} /> : <Radio size={16} />}
            {dash?.proximity_enabled ? 'Proximity on' : 'Enable proximity'}
          </button>
        </div>
      </header>

      <AnalyticsHeader
        todayTotal={dash?.today_total}
        tripCount={dash?.trip_count}
        walletBalance={dash?.wallet_balance}
      />

      <section className="mt-6 grid gap-4 lg:grid-cols-2">
        <FareTiersEditor
          tiers={dash?.fare_tiers}
          onSave={async (fare_tiers) => {
            const res = await fetch(`${API}/api/v1/driver/${vehicle}/fare-tiers`, {
              method: 'PUT',
              headers: { 'Content-Type': 'application/json' },
              body: JSON.stringify({ fare_tiers }),
            })
            const data = await res.json()
            if (!res.ok) throw new Error(data.error || 'Failed to save')
            setDash((d) => (d ? { ...d, fare_tiers: data.fare_tiers } : d))
          }}
        />
        <WithdrawPanel
          balance={dash?.wallet_balance}
          onWithdraw={async (amount) => {
            const res = await fetch(`${API}/api/v1/driver/${vehicle}/payout`, {
              method: 'POST',
              headers: { 'Content-Type': 'application/json' },
              body: JSON.stringify({ amount }),
            })
            const data = await res.json()
            if (!res.ok) throw new Error(data.error || 'Payout failed')
            setDash((d) =>
              d ? { ...d, wallet_balance: data.wallet_balance ?? d.wallet_balance - amount } : d,
            )
          }}
        />
      </section>

      <section className="mt-6">
        <h2 className="mb-3 text-sm font-bold uppercase tracking-wide text-cab-ink/55">
          Recent transactions
        </h2>
        <RecentTransactionsTable transactions={txs} flashIds={flashIds} />
      </section>
    </div>
  )
}
