export default function FareTierPicker({ tiers, selected, onSelect }) {
  if (!tiers?.length) return null

  return (
    <div className="space-y-2">
      <p className="text-sm font-medium text-mpesa-ink/80">Choose fare</p>
      <div className="grid gap-2">
        {tiers.map((tier) => {
          const active = selected?.name === tier.name && selected?.amount === tier.amount
          return (
            <button
              key={`${tier.name}-${tier.amount}`}
              type="button"
              onClick={() => onSelect(tier)}
              className={`flex items-center justify-between rounded-xl border-2 px-4 py-3 text-left transition ${
                active
                  ? 'border-mpesa-green bg-mpesa-light'
                  : 'border-transparent bg-white ring-1 ring-black/5'
              }`}
            >
              <span className="font-semibold text-mpesa-ink">{tier.name}</span>
              <span className="font-mono font-bold text-mpesa-green">
                {Number(tier.amount).toFixed(2)} ETB
              </span>
            </button>
          )
        })}
      </div>
    </div>
  )
}
