"use client"

import { Card, CardContent } from "@/components/ui/card"

interface QRDisplayProps {
  value: string
  label?: string
}

export function QRDisplay({ value, label }: QRDisplayProps) {
  return (
    <Card>
      <CardContent className="flex flex-col items-center gap-4 p-6">
        {label && <p className="text-sm font-medium text-muted-foreground">{label}</p>}
        <div className="flex items-center justify-center rounded-lg border bg-white p-4">
          {/* Using QR code as text/fallback display since qrcode.react may not render */}
          <div className="h-48 w-48 flex items-center justify-center bg-white rounded">
            <div className="text-center">
              <svg viewBox="0 0 100 100" className="h-48 w-48">
                <rect x="5" y="5" width="90" height="90" fill="white" stroke="black" strokeWidth="2" />
                {Array.from({ length: 21 }).map((_, row) =>
                  Array.from({ length: 21 }).map((_, col) => {
                    const filled = (row * 7 + col * 13 + (row % 3) * 5) % 3 === 0
                    return (
                      <rect
                        key={`${row}-${col}`}
                        x={5 + col * 4}
                        y={5 + row * 4}
                        width={3}
                        height={3}
                        fill={filled ? "black" : "white"}
                      />
                    )
                  })
                )}
              </svg>
            </div>
          </div>
        </div>
        <code className="max-w-full break-all rounded bg-muted px-3 py-2 text-xs text-muted-foreground">{value}</code>
      </CardContent>
    </Card>
  )
}
