import { Banknote, Route, Wallet } from 'lucide-react'

export default function AnalyticsHeader({ todayTotal, tripCount, walletBalance }) {
  return (
    <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
      <Stat icon={Banknote} label="Today's income" value={`${Number(todayTotal || 0).toFixed(2)} ETB`} />
      <Stat icon={Route} label="Trips today" value={String(tripCount ?? 0)} />
      <Stat icon={Wallet} label="M-Pesa wallet" value={`${Number(walletBalance || 0).toFixed(2)} ETB`} />
    </div>
  )
}

function Stat({ icon: Icon, label, value }) {
  return (
    <div className="rounded-2xl bg-white p-4 shadow-sm ring-1 ring-black/5">
      <div className="flex items-center gap-2 text-cab-ink/55">
        <Icon size={18} className="text-cab-green" />
        <span className="text-xs font-semibold uppercase tracking-wide">{label}</span>
      </div>
      <p className="mt-2 font-mono text-2xl font-semibold tracking-tight text-cab-ink">{value}</p>
    </div>
  )
}
