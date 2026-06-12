"use client"

import { useState } from "react"
import { AlertCircle, Save } from "lucide-react"
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "@/components/ui/tabs"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { PageHeader } from "@/components/PageHeader"
import { DataTable } from "@/components/DataTable"
import { EmptyState } from "@/components/EmptyState"
import { StatusBadge } from "@/components/StatusBadge"
import { CopyButton } from "@/components/CopyButton"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
  DialogTrigger,
} from "@/components/ui/dialog"
import { MOCK_SETTINGS, MOCK_API_KEYS } from "@/lib/mock-data"
import type { ApiKey } from "@/lib/mock-data"

export default function SettingsPage() {
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(false)
  const [stellarAddress, setStellarAddress] = useState(MOCK_SETTINGS.stellarAddress)
  const [keyDialogOpen, setKeyDialogOpen] = useState(false)
  const [newKeyLabel, setNewKeyLabel] = useState("")
  const [newKeyResult, setNewKeyResult] = useState("")
  const [showKeyModal, setShowKeyModal] = useState(false)

  if (error) {
    return (
      <div className="flex flex-col items-center gap-4 py-20">
        <AlertCircle className="h-12 w-12 text-destructive" />
        <h2 className="text-xl font-semibold">Failed to load settings</h2>
        <Button onClick={() => setError(false)}>Retry</Button>
      </div>
    )
  }

  const handleGenerateKey = (e: React.FormEvent) => {
    e.preventDefault()
    setNewKeyResult(`fg_${newKeyLabel.toLowerCase().replace(/\s+/g, "_")}_${Math.random().toString(36).slice(2, 10)}`)
    setKeyDialogOpen(false)
    setShowKeyModal(true)
    setNewKeyLabel("")
  }

  return (
    <div className="space-y-6">
      <PageHeader title="Settings" />
      <Tabs defaultValue="profile">
        <TabsList>
          <TabsTrigger value="profile">Profile</TabsTrigger>
          <TabsTrigger value="payout">Payout Address</TabsTrigger>
          <TabsTrigger value="deposit">Deposit Info</TabsTrigger>
          <TabsTrigger value="api-keys">API Keys</TabsTrigger>
        </TabsList>

        <TabsContent value="profile">
          <Card>
            <CardHeader><CardTitle className="text-base">Profile</CardTitle></CardHeader>
            <CardContent className="space-y-4">
              {loading ? (
                <div className="space-y-3"><Skeleton className="h-9 w-full" /><Skeleton className="h-9 w-full" /></div>
              ) : (
                <>
                  <div className="space-y-2">
                    <Label>Email</Label>
                    <Input value={MOCK_SETTINGS.email} readOnly className="bg-muted" />
                  </div>
                  <div className="space-y-2">
                    <Label>Role</Label>
                    <Input value={MOCK_SETTINGS.role.charAt(0).toUpperCase() + MOCK_SETTINGS.role.slice(1)} readOnly className="bg-muted" />
                  </div>
                  <Button><Save className="h-4 w-4" />Save</Button>
                </>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="payout">
          <Card>
            <CardHeader><CardTitle className="text-base">Payout Address</CardTitle></CardHeader>
            <CardContent className="space-y-4">
              {loading ? (
                <div className="space-y-3"><Skeleton className="h-9 w-full" /></div>
              ) : (
                <>
                  <div className="space-y-2">
                    <Label htmlFor="stellar">Stellar Address</Label>
                    <Input id="stellar" value={stellarAddress} onChange={(e) => setStellarAddress(e.target.value)} />
                    <p className="text-xs text-muted-foreground">Payouts are sent bi-weekly to this Stellar address.</p>
                  </div>
                  <Button><Save className="h-4 w-4" />Save</Button>
                </>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="deposit">
          <Card>
            <CardHeader><CardTitle className="text-base">Deposit Info</CardTitle></CardHeader>
            <CardContent className="space-y-4">
              {loading ? (
                <div className="space-y-3"><Skeleton className="h-9 w-full" /><Skeleton className="h-20 w-full" /></div>
              ) : (
                <>
                  <div className="space-y-2">
                    <Label>Deposit Memo</Label>
                    <div className="flex items-center gap-2">
                      <Input value={MOCK_SETTINGS.depositMemo} readOnly className="bg-muted font-mono" />
                      <CopyButton text={MOCK_SETTINGS.depositMemo} />
                    </div>
                  </div>
                  <div className="rounded-lg bg-muted p-4">
                    <p className="text-sm text-muted-foreground">{MOCK_SETTINGS.depositInstructions}</p>
                  </div>
                </>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="api-keys">
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle className="text-base">API Keys</CardTitle>
                <Dialog open={keyDialogOpen} onOpenChange={setKeyDialogOpen}>
                  <DialogTrigger asChild>
                    <Button size="sm">Generate New Key</Button>
                  </DialogTrigger>
                  <DialogContent>
                    <DialogHeader>
                      <DialogTitle>Generate API Key</DialogTitle>
                      <DialogDescription>Create a new API key for programmatic access.</DialogDescription>
                    </DialogHeader>
                    <form onSubmit={handleGenerateKey} className="space-y-4">
                      <div className="space-y-2">
                        <Label htmlFor="key-label">Label</Label>
                        <Input id="key-label" placeholder="e.g. Production" value={newKeyLabel} onChange={(e) => setNewKeyLabel(e.target.value)} required />
                      </div>
                      <DialogFooter>
                        <Button type="submit">Generate</Button>
                      </DialogFooter>
                    </form>
                  </DialogContent>
                </Dialog>
              </div>
            </CardHeader>
            <CardContent>
              {loading ? (
                <div className="space-y-3"><Skeleton className="h-8 w-full" /><Skeleton className="h-8 w-full" /></div>
              ) : (
                <DataTable
                  columns={[
                    { key: "label", header: "Label", cell: (k: ApiKey) => k.label },
                    { key: "key", header: "Key", cell: (k: ApiKey) => (
                      <div className="flex items-center gap-1">
                        <span className="font-mono text-xs text-muted-foreground">{k.prefix}••••••••</span>
                        <CopyButton text={k.prefix + "sk_live_" + Math.random().toString(36).slice(2)} />
                      </div>
                    )},
                    { key: "status", header: "Status", cell: (k: ApiKey) => <StatusBadge status={k.status} /> },
                    { key: "created", header: "Created", cell: (k: ApiKey) => k.createdAt },
                  ]}
                  data={MOCK_API_KEYS}
                  loading={loading}
                />
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      <Dialog open={showKeyModal} onOpenChange={setShowKeyModal}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>API Key Generated</DialogTitle>
            <DialogDescription>
              Copy this key now. You won&apos;t be able to see it again.
            </DialogDescription>
          </DialogHeader>
          <div className="flex items-center gap-2 rounded-lg bg-muted p-4">
            <code className="flex-1 break-all text-sm font-mono">{newKeyResult}</code>
            <CopyButton text={newKeyResult} />
          </div>
          <p className="text-xs text-destructive">Make sure to copy your API key now. You won&apos;t be able to see it again!</p>
          <DialogFooter>
            <Button onClick={() => setShowKeyModal(false)}>Done</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
