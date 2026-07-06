"use client"

import { AlertTriangle, X } from "lucide-react"
import { useSyncExternalStore } from "react"
import { Button } from "@/components/ui/button"
import { formatCurrency } from "@/lib/utils"

interface LowBalanceBannerProps {
  balance: number
  visible: boolean
}

const STORAGE_KEY = "low-balance-dismissed"

const storageListeners = new Set<() => void>()

function subscribeToStorage(onStoreChange: () => void) {
  storageListeners.add(onStoreChange)
  window.addEventListener("storage", onStoreChange)
  return () => {
    storageListeners.delete(onStoreChange)
    window.removeEventListener("storage", onStoreChange)
  }
}

function getSnapshot() {
  return !!localStorage.getItem(STORAGE_KEY)
}

function getServerSnapshot() {
  return false
}

function dismiss() {
  localStorage.setItem(STORAGE_KEY, "true")
  storageListeners.forEach((fn) => fn())
}

export function LowBalanceBanner({ balance, visible }: LowBalanceBannerProps) {
  const dismissed = useSyncExternalStore(subscribeToStorage, getSnapshot, getServerSnapshot)

  if (!visible || dismissed) return null

  return (
    <div className="flex items-center gap-3 border border-amber/30 bg-amber/10 px-4 py-2.5 text-sm">
      <AlertTriangle className="size-4 shrink-0 text-amber" />
      <span className="text-amber">
        Balance low ({formatCurrency(balance)}).{" "}
        <a href="/dashboard/deposit" className="font-medium underline underline-offset-2 hover:text-amber/80">
          Deposit now
        </a>
      </span>
      <Button variant="ghost" size="icon-xs" onClick={dismiss} className="ml-auto shrink-0 text-amber/60 hover:text-amber">
        <X className="size-3.5" />
      </Button>
    </div>
  )
}
