import React from 'react'
import { Banknote } from 'lucide-react'

export default function WithdrawPanel({ balance, onWithdraw }) {
  const [amount, setAmount] = React.useState('')
  const [busy, setBusy] = React.useState(false)
  const [msg, setMsg] = React.useState('')

  const submit = async () => {
    const n = Number(amount)
    if (!n || n <= 0) {
      setMsg('Enter a valid amount')
      return
    }
    if (n > Number(balance || 0)) {
      setMsg('More than ledger balance')
      return
    }
    setBusy(true)
    setMsg('')
    try {
      await onWithdraw(n)
      setAmount('')
      setMsg('Payout submitted')
    } catch (e) {
      setMsg(e.message || 'Payout failed')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="rounded-2xl bg-white p-4 shadow-sm ring-1 ring-black/5">
      <div className="mb-3 flex items-center gap-2">
        <Banknote size={18} className="text-cab-green" />
        <h2 className="text-sm font-bold uppercase tracking-wide text-cab-ink/55">
          Withdraw to M-Pesa
        </h2>
      </div>
      <p className="mb-3 text-xs text-cab-ink/50">
        Sends from GuzoPay merchant float to your phone via B2C. Ledger balance:{' '}
        <span className="font-mono font-semibold text-cab-ink">
          {Number(balance || 0).toFixed(2)} ETB
        </span>
      </p>
      <div className="flex flex-wrap gap-2">
        <input
          type="number"
          min="1"
          step="1"
          value={amount}
          onChange={(e) => setAmount(e.target.value)}
          placeholder="Amount ETB"
          className="min-w-[8rem] flex-1 rounded-lg border border-cab-ink/15 bg-cab-sand/40 px-3 py-2 font-mono text-sm outline-none focus:border-cab-green"
        />
        <button
          type="button"
          disabled={busy}
          onClick={submit}
          className="rounded-lg bg-cab-green px-4 py-2 text-sm font-semibold text-white disabled:opacity-50"
        >
          {busy ? 'Sending…' : 'Withdraw'}
        </button>
      </div>
      {msg && <p className="mt-2 text-xs text-cab-ink/60">{msg}</p>}
    </div>
  )
}
