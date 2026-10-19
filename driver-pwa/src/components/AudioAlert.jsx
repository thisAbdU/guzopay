import React from 'react'
import { Volume2, VolumeX } from 'lucide-react'

export default function AudioAlert({ enabled, onToggle, audioRef }) {
  return (
    <button
      type="button"
      onClick={onToggle}
      className={`inline-flex items-center gap-2 rounded-lg border px-3 py-2 text-sm font-semibold ${
        enabled
          ? 'border-cab-green bg-cab-green text-white'
          : 'border-cab-ink/20 bg-white text-cab-ink'
      }`}
    >
      {enabled ? <Volume2 size={16} /> : <VolumeX size={16} />}
      {enabled ? 'Audio on' : 'Enable audio'}
      <audio ref={audioRef} src="/ding.mp3" preload="auto" />
    </button>
  )
}

export function playDing(audioRef) {
  const el = audioRef?.current
  if (!el) return
  el.currentTime = 0
  el.play().catch(() => {})
}
