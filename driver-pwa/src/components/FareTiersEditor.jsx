import React from 'react'
import { Plus, Trash2 } from 'lucide-react'

export default function FareTiersEditor({ tiers, onSave }) {
  const [draft, setDraft] = React.useState([])
  const [saving, setSaving] = React.useState(false)
  const [msg, setMsg] = React.useState('')

  React.useEffect(() => {
    setDraft((tiers || []).map((t) => ({ name: t.name, amount: t.amount })))
  }, [tiers])

  const update = (i, patch) => {
    setDraft((prev) => prev.map((t, idx) => (idx === i ? { ...t, ...patch } : t)))
  }

  const save = async () => {
    const cleaned = draft
      .map((t) => ({ name: String(t.name || '').trim(), amount: Number(t.amount) }))
      .filter((t) => t.name && t.amount > 0)
    if (!cleaned.length) {
      setMsg('Add at least one tier')
      return
    }
    setSaving(true)
    setMsg('')
    try {
      await onSave(cleaned)
      setMsg('Saved')
    } catch (e) {
      setMsg(e.message || 'Save failed')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="rounded-2xl bg-white p-4 shadow-sm ring-1 ring-black/5">
      <div className="mb-3 flex items-center justify-between gap-2">
        <h2 className="text-sm font-bold uppercase tracking-wide text-cab-ink/55">Fare tiers</h2>
        <button
          type="button"
          onClick={() => setDraft((d) => [...d, { name: '', amount: '' }])}
          className="inline-flex items-center gap-1 rounded-lg border border-cab-ink/15 px-2.5 py-1.5 text-xs font-semibold text-cab-ink"
        >
          <Plus size={14} /> Add
        </button>
      </div>

      <div className="space-y-2">
        {draft.map((tier, i) => (
          <div key={i} className="flex items-center gap-2">
            <input
              value={tier.name}
              onChange={(e) => update(i, { name: e.target.value })}
              placeholder="Name"
              className="min-w-0 flex-1 rounded-lg border border-cab-ink/15 bg-cab-sand/40 px-3 py-2 text-sm outline-none focus:border-cab-green"
            />
            <input
              type="number"
              min="0"
              step="0.5"
              value={tier.amount}
              onChange={(e) => update(i, { amount: e.target.value })}
              placeholder="ETB"
              className="w-24 rounded-lg border border-cab-ink/15 bg-cab-sand/40 px-3 py-2 font-mono text-sm outline-none focus:border-cab-green"
            />
            <button
              type="button"
              onClick={() => setDraft((d) => d.filter((_, idx) => idx !== i))}
              className="rounded-lg p-2 text-cab-ink/40 hover:bg-cab-warn/10 hover:text-cab-warn"
              aria-label="Remove tier"
            >
              <Trash2 size={16} />
            </button>
          </div>
        ))}
      </div>

      <div className="mt-3 flex items-center justify-between gap-2">
        <p className="text-xs text-cab-ink/50">{msg}</p>
        <button
          type="button"
          disabled={saving}
          onClick={save}
          className="rounded-lg bg-cab-green px-4 py-2 text-sm font-semibold text-white disabled:opacity-50"
        >
          {saving ? 'Saving…' : 'Save fares'}
        </button>
      </div>
    </div>
  )
}
