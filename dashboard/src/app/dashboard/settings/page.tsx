"use client"

import { useState } from "react"
import { AlertCircle, Save, Copy, Check, Sun, Moon } from "lucide-react"
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "@/components/ui/tabs"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Separator } from "@/components/ui/separator"
import { Skeleton } from "@/components/ui/skeleton"
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
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
  const [saved, setSaved] = useState(false)
  const [keyDialogOpen, setKeyDialogOpen] = useState(false)
  const [newKeyLabel, setNewKeyLabel] = useState("")
  const [newKeyResult, setNewKeyResult] = useState("")
  const [showKeyModal, setShowKeyModal] = useState(false)
  const [theme, setTheme] = useState<"dark" | "light">("dark")

  if (error) {
    return (
      <div className="flex flex-col items-center gap-4 py-20">
        <AlertCircle className="h-12 w-12 text-destructive" />
        <h2 className="text-xl font-semibold">Failed to load settings</h2>
        <Button onClick={() => setError(false)}>Retry</Button>
      </div>
    )
  }

  const handleSavePayout = () => {
    setSaved(true)
    setTimeout(() => setSaved(false), 2000)
  }

  const handleGenerateKey = (e: React.FormEvent) => {
    e.preventDefault()
    const key = `fg_${newKeyLabel.toLowerCase().replace(/\s+/g, "_")}_${Math.random().toString(36).slice(2, 14)}`
    setNewKeyResult(key)
    setKeyDialogOpen(false)
    setShowKeyModal(true)
    setNewKeyLabel("")
  }

  const toggleTheme = () => {
    const next = theme === "dark" ? "light" : "dark"
    setTheme(next)
    document.documentElement.classList.toggle("dark", next === "dark")
  }

  return (
    <div className="space-y-6">
      <PageHeader title="Settings" description="Manage your account and preferences." />
      <Tabs defaultValue="profile">
        <TabsList className="w-full sm:w-auto">
          <TabsTrigger value="profile">Profile</TabsTrigger>
          <TabsTrigger value="payout">Payout Address</TabsTrigger>
          <TabsTrigger value="deposit">Deposit Info</TabsTrigger>
          <TabsTrigger value="api-keys">API Keys</TabsTrigger>
          <TabsTrigger value="notifications" disabled>Notifications</TabsTrigger>
        </TabsList>

        <TabsContent value="profile" className="mt-6">
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Profile</CardTitle>
              <CardDescription>Your account information from OAuth.</CardDescription>
            </CardHeader>
            <CardContent className="space-y-6">
              {loading ? (
                <div className="space-y-4"><Skeleton className="h-16 w-16 rounded-full" /><Skeleton className="h-9 w-full" /><Skeleton className="h-9 w-full" /></div>
              ) : (
                <>
                  <div className="flex items-center gap-4">
                    <Avatar className="h-16 w-16">
                      <AvatarFallback className="text-lg">CA</AvatarFallback>
                    </Avatar>
                    <div>
                      <p className="font-medium">{MOCK_SETTINGS.email}</p>
                      <p className="text-xs text-muted-foreground">Connected with Google</p>
                    </div>
                  </div>

                  <div className="space-y-2">
                    <Label>Email</Label>
                    <Input value={MOCK_SETTINGS.email} readOnly className="bg-muted" />
                  </div>

                  <div className="space-y-2">
                    <Label>Roles</Label>
                    <div className="flex gap-2">
                      <Badge variant="secondary" className="capitalize">Provider</Badge>
                      <Badge variant="secondary" className="capitalize">Consumer</Badge>
                    </div>
                  </div>

                  <Separator />

                  <div className="space-y-3">
                    <Label>Theme</Label>
                    <div className="flex gap-2">
                      <Button
                        variant={theme === "dark" ? "default" : "outline"}
                        size="sm"
                        onClick={() => { setTheme("dark"); document.documentElement.classList.add("dark") }}
                        className="gap-2"
                      >
                        <Moon className="h-4 w-4" /> Dark
                      </Button>
                      <Button
                        variant={theme === "light" ? "default" : "outline"}
                        size="sm"
                        onClick={() => { setTheme("light"); document.documentElement.classList.remove("dark") }}
                        className="gap-2"
                      >
                        <Sun className="h-4 w-4" /> Light
                      </Button>
                    </div>
                  </div>
                </>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="payout" className="mt-6">
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Payout Address</CardTitle>
              <CardDescription>Set your Stellar wallet address for receiving payouts.</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              {loading ? (
                <Skeleton className="h-9 w-full" />
              ) : (
                <>
                  <div className="space-y-2">
                    <Label htmlFor="stellar">Stellar Address</Label>
                    <Input id="stellar" value={stellarAddress} onChange={(e) => setStellarAddress(e.target.value)} className="font-mono text-sm" />
                    <p className="text-xs text-muted-foreground">
                      Payouts are processed bi-weekly. Address must be a valid Stellar account.
                    </p>
                  </div>
                  <Button onClick={handleSavePayout}>
                    {saved ? <><Check className="mr-1.5 h-4 w-4" /> Saved</> : <><Save className="mr-1.5 h-4 w-4" /> Save</>}
                  </Button>
                </>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="deposit" className="mt-6">
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Deposit Info</CardTitle>
              <CardDescription>Your deposit details for receiving XLM.</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              {loading ? (
                <div className="space-y-3"><Skeleton className="h-9 w-full" /><Skeleton className="h-9 w-full" /></div>
              ) : (
                <>
                  <div className="space-y-2">
                    <Label>Deposit Memo</Label>
                    <div className="flex items-center gap-2">
                      <Input value={MOCK_SETTINGS.depositMemo} readOnly className="bg-muted font-mono" />
                      <CopyButton text={MOCK_SETTINGS.depositMemo} />
                    </div>
                  </div>
                  <div className="space-y-2">
                    <Label>Deposit Address</Label>
                    <div className="flex items-center gap-2">
                      <Input value={MOCK_SETTINGS.stellarAddress} readOnly className="bg-muted font-mono text-xs" />
                      <CopyButton text={MOCK_SETTINGS.stellarAddress} />
                    </div>
                  </div>
                  <div className="rounded-lg bg-warning/10 border border-warning/30 p-3">
                    <p className="text-xs text-warning">Minimum deposit: 5 XLM</p>
                  </div>
                </>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="api-keys" className="mt-6">
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <div>
                  <CardTitle className="text-base">API Keys</CardTitle>
                  <CardDescription>Manage your API keys for authentication.</CardDescription>
                </div>
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

        <TabsContent value="notifications" className="mt-6">
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Notifications</CardTitle>
              <CardDescription>Configure notification preferences (coming soon).</CardDescription>
            </CardHeader>
            <CardContent>
              <EmptyState
                title="Coming soon"
                description="Email and notification preferences will be available in a future update."
              />
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
          <div className="rounded-lg border border-destructive/30 bg-destructive/10 p-3">
            <p className="text-xs text-destructive font-medium">
              This key will not be shown again after you close this dialog.
            </p>
          </div>
          <DialogFooter>
            <Button onClick={() => setShowKeyModal(false)}>Done</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
