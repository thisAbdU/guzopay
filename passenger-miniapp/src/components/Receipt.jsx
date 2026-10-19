export default function Receipt({ receipt, onDone }) {
  return (
    <div className="flex min-h-[70dvh] flex-col items-center justify-center px-2 text-center">
      <div className="animate-check flex h-20 w-20 items-center justify-center rounded-full bg-mpesa-green text-4xl text-white shadow-lg shadow-mpesa-green/30">
        ✓
      </div>
      <h2 className="mt-5 text-2xl font-bold text-mpesa-ink">Payment successful</h2>
      <p className="mt-1 text-sm text-mpesa-ink/60">Fare paid via M-Pesa</p>

      <div className="mt-8 w-full rounded-2xl bg-white p-5 text-left shadow-sm ring-1 ring-black/5">
        <Row label="Transaction ID" value={receipt.mpesa_tx_id} mono />
        <Row label="Vehicle" value={`#${receipt.vehicle_code}`} />
        <Row label="Driver" value={receipt.driver_name} />
        {receipt.fare_tier_name && <Row label="Fare" value={receipt.fare_tier_name} />}
        <Row label="Amount" value={`${Number(receipt.amount).toFixed(2)} ETB`} />
        <Row label="Time" value={new Date(receipt.created_at).toLocaleString()} />
      </div>

      <button
        type="button"
        onClick={onDone}
        className="mt-8 w-full rounded-xl bg-mpesa-dark py-3.5 font-semibold text-white"
      >
        Done
      </button>
    </div>
  )
}

function Row({ label, value, mono }) {
  return (
    <div className="flex items-start justify-between gap-3 border-b border-black/5 py-2.5 last:border-0">
      <span className="text-sm text-mpesa-ink/55">{label}</span>
      <span className={`text-right text-sm font-semibold text-mpesa-ink ${mono ? 'font-mono text-xs' : ''}`}>
        {value}
      </span>
    </div>
  )
}
