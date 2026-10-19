import React from 'react'
import { Radio, RadioTower } from 'lucide-react'
import AnalyticsHeader from './components/AnalyticsHeader.jsx'
import AudioAlert, { playDing } from './components/AudioAlert.jsx'
import FareTiersEditor from './components/FareTiersEditor.jsx'
import RecentTransactionsTable from './components/RecentTransactionsTable.jsx'

const API = import.meta.env.VITE_API_URL || 'http://localhost:8080'
const WS = import.meta.env.VITE_WS_URL || 'ws://localhost:8080'
const VEHICLE = import.meta.env.VITE_VEHICLE_CODE || '4A2Q'

export default function App() {
  const [dash, setDash] = React.useState(null)
  const [txs, setTxs] = React.useState([])
  const [flashIds, setFlashIds] = React.useState(() => new Set())
  const [audioOn, setAudioOn] = React.useState(false)
  const [wsState, setWsState] = React.useState('connecting')
  const audioRef = React.useRef(null)
  const audioOnRef = React.useRef(false)

  React.useEffect(() => {
    audioOnRef.current = audioOn
  }, [audioOn])

  const loadDashboard = React.useCallback(async () => {
    const res = await fetch(`${API}/api/v1/driver/${VEHICLE}/dashboard`)
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || 'Failed to load')
    setDash(data)
    setTxs(data.transactions || [])
  }, [])

  React.useEffect(() => {
    loadDashboard().catch(console.error)
  }, [loadDashboard])

  React.useEffect(() => {
    let ws
    let closed = false
    let retry

    const connect = () => {
      setWsState('connecting')
      ws = new WebSocket(`${WS}/ws/driver/${VEHICLE}`)
      ws.onopen = () => setWsState('live')
      ws.onclose = () => {
        setWsState('offline')
        if (!closed) retry = setTimeout(connect, 2000)
      }
      ws.onerror = () => ws.close()
      ws.onmessage = (ev) => {
        try {
          const msg = JSON.parse(ev.data)
          if (msg.type !== 'payment' || !msg.payment) return
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
  }, [])

  const toggleProximity = async () => {
    if (!dash) return
    const enabled = !dash.proximity_enabled
    const res = await fetch(`${API}/api/v1/driver/${VEHICLE}/proximity`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ enabled }),
    })
    const data = await res.json()
    if (res.ok) setDash((d) => ({ ...d, proximity_enabled: data.proximity_enabled }))
  }

  return (
    <div className="mx-auto min-h-dvh max-w-5xl px-4 py-5 sm:px-6">
      <header className="mb-5 flex flex-wrap items-start justify-between gap-3">
        <div>
          <p className="text-xs font-bold uppercase tracking-[0.18em] text-cab-green">GuzoPay Driver</p>
          <h1 className="text-2xl font-bold text-cab-ink sm:text-3xl">
            Taxi #{VEHICLE}
            {dash?.driver_name ? ` · ${dash.driver_name}` : ''}
          </h1>
          <p className="mt-1 text-sm text-cab-ink/60">{dash?.route || 'Loading route…'}</p>
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

      <section className="mt-6">
        <FareTiersEditor
          tiers={dash?.fare_tiers}
          onSave={async (fare_tiers) => {
            const res = await fetch(`${API}/api/v1/driver/${VEHICLE}/fare-tiers`, {
              method: 'PUT',
              headers: { 'Content-Type': 'application/json' },
              body: JSON.stringify({ fare_tiers }),
            })
            const data = await res.json()
            if (!res.ok) throw new Error(data.error || 'Failed to save')
            setDash((d) => (d ? { ...d, fare_tiers: data.fare_tiers } : d))
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
