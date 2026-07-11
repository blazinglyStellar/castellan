"use client"

import { ApiKeysView } from "@/components/api-keys/api-keys-view"

export default function ApiKeysPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">API Keys</h1>
      </div>
      <ApiKeysView />
    </div>
  )
}
