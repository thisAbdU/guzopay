import React from 'react'
import CodeEntry from './components/CodeEntry.jsx'
import FareTierPicker from './components/FareTierPicker.jsx'
import ProximityDetect from './components/ProximityDetect.jsx'
import Receipt from './components/Receipt.jsx'
import {
  alert as mpesaAlert,
  getAuthCode,
  hideLoading,
  isMiniProgram,
  setNavigationBar,
  showLoading,
  showToast,
  tradePay,
} from './mpesa/bridge.js'

const API = import.meta.env.VITE_API_URL || 'http://localhost:8080'

export default function App() {
  const [code, setCode] = React.useState('')
  const [taxi, setTaxi] = React.useState(null)
  const [selectedTier, setSelectedTier] = React.useState(null)
  const [error, setError] = React.useState('')
  const [paying, setPaying] = React.useState(false)
  const [receipt, setReceipt] = React.useState(null)
  const [passenger, setPassenger] = React.useState({
    passenger_name: 'Abebe Kebede',
    passenger_phone: '0712345678',
  })
  const [authCode, setAuthCode] = React.useState(null)
  const inMiniApp = isMiniProgram()

  React.useEffect(() => {
    setNavigationBar('GuzoPay')
    ;(async () => {
      try {
        const auth = await getAuthCode(['auth_user'])
        if (!auth?.authCode) return
        setAuthCode(auth.authCode)
        const res = await fetch(`${API}/api/v1/mpesa/auth`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ auth_code: auth.authCode }),
        })
        const data = await res.json()
        if (res.ok && data.passenger_name) {
          setPassenger({
            passenger_name: data.passenger_name,
            passenger_phone: data.passenger_phone || '0712345678',
          })
        }
      } catch {
        // Browser / unauthorized — keep demo passenger
      }
    })()
  }, [])

  React.useEffect(() => {
    const tiers = taxi?.fare_tiers || []
    if (!tiers.length) {
      setSelectedTier(null)
      return
    }
    setSelectedTier((prev) => {
      if (prev && tiers.some((t) => t.name === prev.name && t.amount === prev.amount)) return prev
      return tiers[tiers.length - 1]
    })
  }, [taxi])

  const selectNearby = (vehicleCode) => {
    setCode(vehicleCode)
  }

  const pay = async () => {
    if (!taxi || !selectedTier) return
    setPaying(true)
    setError('')
    try {
      await showLoading('Processing payment…')

      if (inMiniApp) {
        // Official flow: create order → my.tradePay(tradeNO) → confirm / notifyPayment
        const initRes = await fetch(`${API}/api/v1/pay/initiate`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            vehicle_code: taxi.vehicle_code,
            amount: selectedTier.amount,
            fare_tier_name: selectedTier.name,
            passenger_name: passenger.passenger_name,
            passenger_phone: passenger.passenger_phone,
            auth_code: authCode,
          }),
        })
        const order = await initRes.json()
        if (!initRes.ok) throw new Error(order.error || 'Could not create payment order')

        let payRes
        try {
          payRes = await tradePay({ tradeNO: order.tradeNO })
        } catch (e) {
          // Sandbox / mock tradeNO: wallet may reject — confirm locally for hackathon demos
          if (order.mock) {
            payRes = { resultCode: '9000' }
          } else {
            throw e
          }
        }

        const resultCode = String(payRes?.resultCode || '')
        if (resultCode === '6001') {
          await showToast('Payment cancelled')
          throw new Error('Payment cancelled')
        }
        if (resultCode !== '9000' && resultCode !== '8000' && resultCode !== '6004') {
          throw new Error(`Payment failed (${resultCode || 'unknown'})`)
        }

        const confRes = await fetch(`${API}/api/v1/pay/confirm`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            payment_id: order.payment_id,
            tradeNO: order.tradeNO,
            result_code: resultCode === '8000' ? '9000' : resultCode,
          }),
        })
        const data = await confRes.json()
        if (!confRes.ok) throw new Error(data.error || 'Confirm failed')

        setReceipt({
          mpesa_tx_id: data.mpesa_tx_id,
          vehicle_code: data.vehicle_code,
          driver_name: data.driver_name,
          amount: data.payment.amount,
          fare_tier_name: data.payment.fare_tier_name || selectedTier.name,
          created_at: data.payment.created_at,
        })
        await showToast('Payment successful')
      } else {
        // Browser / IDE preview fallback (no JSBridge)
        const res = await fetch(`${API}/api/v1/pay`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            vehicle_code: taxi.vehicle_code,
            amount: selectedTier.amount,
            fare_tier_name: selectedTier.name,
            passenger_name: passenger.passenger_name,
            passenger_phone: passenger.passenger_phone,
          }),
        })
        const data = await res.json()
        if (!res.ok) throw new Error(data.error || 'Payment failed')
        setReceipt({
          mpesa_tx_id: data.mpesa_tx_id,
          vehicle_code: data.vehicle_code,
          driver_name: data.driver_name,
          amount: data.payment.amount,
          fare_tier_name: data.payment.fare_tier_name || selectedTier.name,
          created_at: data.payment.created_at,
        })
      }
    } catch (e) {
      const msg = e.message || 'Payment failed'
      setError(msg)
      if (inMiniApp) await mpesaAlert({ title: 'Payment', content: msg })
    } finally {
      await hideLoading()
      setPaying(false)
    }
  }

  if (receipt) {
    return (
      <Shell inMiniApp={inMiniApp}>
        <Receipt
          receipt={receipt}
          onDone={() => {
            setReceipt(null)
            setCode('')
            setTaxi(null)
            setSelectedTier(null)
          }}
        />
      </Shell>
    )
  }

  return (
    <Shell inMiniApp={inMiniApp}>
      <header className="mb-6">
        <p className="text-xs font-semibold uppercase tracking-[0.2em] text-mpesa-green">
          {inMiniApp ? 'M-Pesa Mini App' : 'M-Pesa · GuzoPay (preview)'}
        </p>
        <h1 className="mt-1 text-2xl font-bold text-mpesa-ink">Pay your fare</h1>
        <p className="mt-1 text-sm text-mpesa-ink/65">
          Enter the taxi code or tap a nearby vehicle.
        </p>
        {passenger.passenger_name && (
          <p className="mt-2 text-xs text-mpesa-ink/50">Paying as {passenger.passenger_name}</p>
        )}
      </header>

      <div className="space-y-5">
        <ProximityDetect onSelect={selectNearby} />
        <CodeEntry
          code={code}
          onCodeChange={setCode}
          taxi={taxi}
          onTaxi={setTaxi}
          error={error}
          onError={setError}
        />

        {taxi && (
          <FareTierPicker
            tiers={taxi.fare_tiers}
            selected={selectedTier}
            onSelect={setSelectedTier}
          />
        )}

        <button
          type="button"
          disabled={!taxi || !selectedTier || paying}
          onClick={pay}
          className="w-full rounded-xl bg-mpesa-green py-4 text-base font-bold text-white shadow-md shadow-mpesa-green/25 disabled:cursor-not-allowed disabled:opacity-40"
        >
          {paying
            ? 'Processing…'
            : selectedTier
              ? `Pay ${Number(selectedTier.amount).toFixed(2)} ETB via M-Pesa`
              : 'Select a fare'}
        </button>
      </div>
    </Shell>
  )
}

function Shell({ children, inMiniApp }) {
  return (
    <div className="mx-auto flex min-h-dvh max-w-md flex-col px-4 pb-10 pt-6">
      {!inMiniApp && (
        <div className="mb-4 flex items-center gap-2 text-mpesa-dark">
          <span className="inline-flex h-8 w-8 items-center justify-center rounded-lg bg-mpesa-green text-sm font-bold text-white">
            M
          </span>
          <span className="text-sm font-semibold">Safaricom M-Pesa</span>
        </div>
      )}
      {children}
    </div>
  )
}
