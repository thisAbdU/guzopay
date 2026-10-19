export default function RecentTransactionsTable({ transactions, flashIds }) {
  if (!transactions?.length) {
    return (
      <div className="rounded-2xl bg-white px-4 py-10 text-center text-sm text-cab-ink/55 shadow-sm ring-1 ring-black/5">
        Waiting for passenger payments…
      </div>
    )
  }

  return (
    <div className="overflow-hidden rounded-2xl bg-white shadow-sm ring-1 ring-black/5">
      <div className="overflow-x-auto">
        <table className="min-w-full text-left text-sm">
          <thead className="bg-cab-sand text-xs uppercase tracking-wide text-cab-ink/55">
            <tr>
              <th className="px-4 py-3 font-semibold">Time</th>
              <th className="px-4 py-3 font-semibold">Customer</th>
              <th className="px-4 py-3 font-semibold">Phone</th>
              <th className="px-4 py-3 font-semibold">Amount</th>
              <th className="px-4 py-3 font-semibold">Status</th>
            </tr>
          </thead>
          <tbody>
            {transactions.map((tx) => (
              <tr
                key={tx.id}
                className={`border-t border-black/5 ${flashIds?.has(tx.id) ? 'flash-new' : ''}`}
              >
                <td className="px-4 py-3 font-mono text-cab-ink/80">
                  {formatTime(tx.created_at)}
                </td>
                <td className="px-4 py-3 font-medium text-cab-ink">{tx.passenger_name}</td>
                <td className="px-4 py-3 font-mono text-cab-ink/70">{tx.passenger_phone}</td>
                <td className="px-4 py-3 font-mono font-semibold">
                  {Number(tx.amount).toFixed(2)} ETB
                </td>
                <td className="px-4 py-3">
                  <span className="inline-flex rounded-md bg-cab-lime/15 px-2 py-1 text-xs font-bold text-cab-green">
                    Verified
                  </span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}

function formatTime(iso) {
  try {
    return new Date(iso).toLocaleTimeString('en-GB', { hour12: false })
  } catch {
    return '—'
  }
}
