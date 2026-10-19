import React from 'react'
import CodeEntry from './components/CodeEntry.jsx'
import FareTierPicker from './components/FareTierPicker.jsx'
import FeaturePhonePay from './components/FeaturePhonePay.jsx'
import ProximityDetect from './components/ProximityDetect.jsx'
import Receipt from './components/Receipt.jsx'
import {
  alert as mpesaAlert,
  getUserScopes,
  hideLoading,
  isMiniProgram,
  payWithMpesa,
  setNavigationBar,
  showLoading,
  showToast,
} from './mpesa/bridge.js'

const API = import.meta.env.VITE_API_URL || 'http://localhost:8080'
const BUSINESS_ID = import.meta.env.VITE_MPESA_BUSINESS_ID || '600978'

export default function App() {
  const [code, setCode] = React.useState('')
  const [taxi, setTaxi] = React.useState(null)
  const [selectedTier, setSelectedTier] = React.useState(null)
  const [error, setError] = React.useState('')
  const [paying, setPaying] = React.useState(false)
  const [receipt, setReceipt] = React.useState(null)
  // Filled silently from M-Pesa userScopes (SIM) — never typed by the user.
  const passengerRef = React.useRef({ name: '', phone: '' })
  const [profileReady, setProfileReady] = React.useState(!isMiniProgram())
  const inMiniApp = isMiniProgram()

  React.useEffect(() => {
    setNavigationBar('GuzoPay')
    ;(async () => {
      if (!inMiniApp) {
        setProfileReady(true)
        return
      }
      try {
        const profile = await getUserScopes()
        passengerRef.current = {
          name: profile.passenger_name || '',
          phone: profile.passenger_phone || '',
        }
      } catch (e) {
        console.warn('userScopes failed', e)
      } finally {
        setProfileReady(true)
      }
    })()
  }, [inMiniApp])

  React.useEffect(() => {
    const tiers = taxi?.fare_tiers || []
    if (!tiers.length) {
      setSelectedTier(null)
      return
    }
    // One tier → auto-select for true one-tap; else keep last / Full Route default.
    if (tiers.length === 1) {
      setSelectedTier(tiers[0])
      return
    }
    setSelectedTier((prev) => {
      if (prev && tiers.some((t) => t.name === prev.name && t.amount === prev.amount)) return prev
      return tiers[tiers.length - 1]
    })
  }, [taxi])

  const pay = async () => {
    if (!taxi || !selectedTier) return
    setPaying(true)
    setError('')
    const { name, phone } = passengerRef.current
    const payload = {
      vehicle_code: taxi.vehicle_code,
      amount: selectedTier.amount,
      fare_tier_name: selectedTier.name,
      passenger_name: name,
      passenger_phone: phone,
    }
    try {
      await showLoading('Processing payment…')

      if (inMiniApp) {
        // Wallet charges the logged-in M-Pesa SIM; scopes only label the receipt.
        const initRes = await fetch(`${API}/api/v1/pay/initiate`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(payload),
        })
        const order = await initRes.json()
        if (!initRes.ok) throw new Error(order.error || 'Could not create payment order')

        let payRes
        try {
          payRes = await payWithMpesa({
            businessID: BUSINESS_ID,
            amount: selectedTier.amount,
            billReference: taxi.vehicle_code,
            reason: `Fare · Taxi #${taxi.vehicle_code}`,
          })
        } catch (e) {
          if (e?.error === 201) {
            await showToast('Payment cancelled')
            throw new Error('Payment cancelled')
          }
          throw e
        }

        const txId = payRes?.transactionId || payRes?.transactionID || ''
        if (!txId) {
          const done = await pollPayment(order.payment_id, 60000)
          setReceipt(receiptFromPoll(done, selectedTier.name))
          await showToast('Payment successful')
          return
        }

        const confRes = await fetch(`${API}/api/v1/pay/confirm`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            payment_id: order.payment_id,
            tradeNO: txId,
            result_code: '9000',
          }),
        })
        const data = await confRes.json()
        if (!confRes.ok) {
          try {
            const done = await pollPayment(order.payment_id, 30000)
            setReceipt(receiptFromPoll(done, selectedTier.name))
            await showToast('Payment successful')
            return
          } catch {
            throw new Error(data.error || 'Confirm failed')
          }
        }

        setReceipt({
          mpesa_tx_id: data.mpesa_tx_id || txId,
          vehicle_code: data.vehicle_code,
          driver_name: data.driver_name,
          amount: data.payment.amount,
          fare_tier_name: data.payment.fare_tier_name || selectedTier.name,
          created_at: data.payment.created_at,
        })
        await showToast('Payment successful')
      } else {
        // Browser preview only (no SIM) — Daraja simulate / mock
        const initRes = await fetch(`${API}/api/v1/pay/daraja`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(payload),
        })
        const order = await initRes.json()

        if (!initRes.ok || order.mock) {
          const res = await fetch(`${API}/api/v1/pay`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload),
          })
          const data = await res.json()
          if (!res.ok) throw new Error(data.error || order.error || 'Payment failed')
          setReceipt({
            mpesa_tx_id: data.mpesa_tx_id,
            vehicle_code: data.vehicle_code,
            driver_name: data.driver_name,
            amount: data.payment.amount,
            fare_tier_name: data.payment.fare_tier_name || selectedTier.name,
            created_at: data.payment.created_at,
          })
        } else {
          const done = await pollPayment(order.payment_id, 45000)
          setReceipt(receiptFromPoll(done, selectedTier.name))
        }
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

  const tiers = taxi?.fare_tiers || []
  const showTierPicker = tiers.length > 1
  const canPay = Boolean(taxi && selectedTier && profileReady)

  return (
    <Shell inMiniApp={inMiniApp}>
      <header className="mb-6">
        <p className="text-xs font-semibold uppercase tracking-[0.2em] text-mpesa-green">
          {inMiniApp ? 'M-Pesa Mini App' : 'GuzoPay (preview)'}
        </p>
        <h1 className="mt-1 text-2xl font-bold text-mpesa-ink">Pay your fare</h1>
        <p className="mt-1 text-sm text-mpesa-ink/65">
          Enter the taxi code and tap pay. Charged from your M-Pesa line.
        </p>
      </header>

      <div className="space-y-5">
        <ProximityDetect onSelect={setCode} />
        <CodeEntry
          code={code}
          onCodeChange={setCode}
          taxi={taxi}
          onTaxi={setTaxi}
          error={error}
          onError={setError}
        />

        {showTierPicker && (
          <FareTierPicker
            tiers={tiers}
            selected={selectedTier}
            onSelect={setSelectedTier}
          />
        )}

        {taxi && selectedTier && !inMiniApp && (
          <FeaturePhonePay
            businessID={BUSINESS_ID}
            vehicleCode={taxi.vehicle_code}
            amount={selectedTier.amount}
            tierName={selectedTier.name}
          />
        )}

        <button
          type="button"
          disabled={!canPay || paying}
          onClick={pay}
          className="w-full rounded-xl bg-mpesa-green py-4 text-base font-bold text-white shadow-md shadow-mpesa-green/25 disabled:cursor-not-allowed disabled:opacity-40"
        >
          {paying
            ? 'Processing…'
            : !taxi
              ? 'Enter taxi code'
              : selectedTier
                ? `Pay ${Number(selectedTier.amount).toFixed(2)} ETB`
                : 'Select a fare'}
        </button>
      </div>
    </Shell>
  )
}

function receiptFromPoll(done, tierName) {
  return {
    mpesa_tx_id: done.mpesa_tx_id || done.payment?.mpesa_tx_id,
    vehicle_code: done.vehicle_code,
    driver_name: done.driver_name,
    amount: done.payment.amount,
    fare_tier_name: done.payment.fare_tier_name || tierName,
    created_at: done.payment.created_at,
  }
}

async function pollPayment(paymentId, timeoutMs) {
  const start = Date.now()
  while (Date.now() - start < timeoutMs) {
    const res = await fetch(`${API}/api/v1/pay/${paymentId}`)
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || 'Payment lookup failed')
    if (data.payment?.status === 'COMPLETED') return data
    if (data.payment?.status === 'FAILED') throw new Error('Payment failed')
    await new Promise((r) => setTimeout(r, 1500))
  }
  throw new Error('Timed out waiting for M-Pesa confirmation.')
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
