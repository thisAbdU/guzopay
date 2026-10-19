/**
 * Feature / "torch" phones cannot run the Mini App.
 * Commuter pays via M-Pesa Pay Bill using the merchant shortcode + vehicle BillRef.
 */
export default function FeaturePhonePay({ businessID, vehicleCode, amount, tierName }) {
  if (!vehicleCode || !amount) return null

  return (
    <div className="rounded-2xl border border-dashed border-mpesa-ink/20 bg-white/80 px-4 py-4">
      <p className="text-xs font-semibold uppercase tracking-wide text-mpesa-ink/55">
        No smartphone? (feature / torch phone)
      </p>
      <p className="mt-1 text-sm text-mpesa-ink/80">
        Pay with M-Pesa Pay Bill from any Safaricom line:
      </p>
      <ul className="mt-3 space-y-1.5 font-mono text-sm text-mpesa-ink">
        <li>
          Business / Till: <strong>{businessID}</strong>
        </li>
        <li>
          Account / Bill Ref: <strong>{vehicleCode}</strong>
        </li>
        <li>
          Amount: <strong>{Number(amount).toFixed(2)} ETB</strong>
          {tierName ? ` (${tierName})` : ''}
        </li>
      </ul>
      <p className="mt-3 text-xs text-mpesa-ink/50">
        After payment, the driver dashboard updates when Safaricom sends the C2B confirmation
        (same shortcode + Bill Ref = taxi code).
      </p>
    </div>
  )
}
