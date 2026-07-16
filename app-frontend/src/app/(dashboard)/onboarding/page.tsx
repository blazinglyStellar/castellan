"use client"

import { useState, useEffect, useRef } from "react"
import { useRouter } from "next/navigation"
import { useQuery, useMutation } from "@tanstack/react-query"
import { copyToClipboard } from "@/lib/clipboard"
import {
  Wallet,
  Copy,
  Check,
  ArrowRight,
  ArrowLeft,
  PartyPopper,
  Loader2,
  Key,
  Search,
  Terminal,
  Server,
  Shield,
  Star,
} from "lucide-react"

import { useAuth } from "@/lib/auth/auth-context"
import {
  getApiConfig,
  getDepositIntent,
  getDiscoverProviders,
  getPublicProviderEndpoints,
  createApiKey,
  completeOnboarding,
  getBalance,
} from "@/lib/api/endpoints"
import type {
  ApiConfigResponse,
  Provider,
  Endpoint,
  CreateApiKeyResponse,
  IntentResponse,
} from "@/lib/api/types"
import { Card, CardContent } from "@/components/ui/card"
import { Button, buttonVariants } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Skeleton } from "@/components/ui/skeleton"
import { MethodBadge } from "@/components/usage/method-badge"
import { getStellarNetworkLabel } from "@/lib/stellar"

const STEPS = [
  {
    id: "welcome",
    title: "Welcome to Castellan",
    icon: Star,
    description: "Usage-based API monetization on Stellar",
  },
  {
    id: "deposit",
    title: "Deposit Funds",
    icon: Wallet,
    description: "Add XLM to your account to start using APIs",
  },
  {
    id: "api-key",
    title: "Create an API Key",
    icon: Key,
    description: "Generate a key to authenticate your requests",
  },
  {
    id: "discover",
    title: "Test Your First API",
    icon: Terminal,
    description: "Try the httpbin demo provider",
  },
  {
    id: "finish",
    title: "You're All Set",
    icon: PartyPopper,
    description: "Start publishing your own APIs",
  },
]

function StepIndicator({ current, total }: { current: number; total: number }) {
  return (
    <div className="mb-8 flex items-center justify-center gap-2">
      {Array.from({ length: total }, (_, i) => (
        <div
          key={i}
          className={`h-2 rounded-full transition-all duration-300 ${
            i === current
              ? "w-8 bg-primary"
              : i < current
                ? "w-2 bg-primary/40"
                : "w-2 bg-muted-foreground/20"
          }`}
        />
      ))}
    </div>
  )
}

export default function OnboardingPage() {
  const { user, updateUser } = useAuth()
  const router = useRouter()
  const [step, setStep] = useState(0)
  const [apiKeyLabel, setApiKeyLabel] = useState("My First Key")
  const [createdKey, setCreatedKey] = useState<CreateApiKeyResponse | null>(null)
  const [keyCopied, setKeyCopied] = useState(false)
  const [depositConfirmed, setDepositConfirmed] = useState(false)
  const [pollingDeposit, setPollingDeposit] = useState(false)
  const [pollingSeconds, setPollingSeconds] = useState(0)
  const pollingIntervalRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const [createKeyError, setCreateKeyError] = useState("")
  const [completeError, setCompleteError] = useState("")
  const [discoverError, setDiscoverError] = useState(false)

  const httpbinName = "httpbin"

  const { data: depositData } = useQuery<IntentResponse>({
    queryKey: ["onboarding-deposit-intent"],
    queryFn: getDepositIntent,
  })

  const { data: discoverData, isLoading: discoverLoading } = useQuery({
    queryKey: ["onboarding-discover"],
    queryFn: getDiscoverProviders,
  })

  const httpbinProvider = (discoverData?.data ?? []).find(
    (p) => p.name.toLowerCase() === httpbinName,
  )

  useEffect(() => {
    if (!discoverLoading && discoverData && !httpbinProvider) {
      const timer = setTimeout(() => setDiscoverError(true), 3000)
      return () => clearTimeout(timer)
    }
  }, [discoverLoading, discoverData, httpbinProvider])

  const { data: httpbinEndpoints } = useQuery({
    queryKey: ["onboarding-httpbin-endpoints", httpbinProvider?.id],
    queryFn: () =>
      httpbinProvider
        ? getPublicProviderEndpoints(httpbinProvider.id)
        : Promise.resolve([]),
    enabled: !!httpbinProvider,
  })

  const createKeyMutation = useMutation({
    mutationFn: (label: string) => createApiKey(label),
    onSuccess: (data) => {
      setCreatedKey(data)
      setCreateKeyError("")
    },
    onError: () => setCreateKeyError("Failed to create API key. Please try again."),
  })

  const completeMutation = useMutation({
    mutationFn: completeOnboarding,
    onSuccess: () => {
      updateUser({ onboarding_completed: true })
      router.replace("/overview")
    },
    onError: () => setCompleteError("Something went wrong. Please try again."),
  })

  useEffect(() => {
    if (pollingDeposit) {
      pollingIntervalRef.current = setInterval(async () => {
        setPollingSeconds((s) => s + 1)
        try {
          const bal = await getBalance()
          if (bal.balance !== "0") {
            setDepositConfirmed(true)
            setPollingDeposit(false)
          }
        } catch {
          // ignore polling errors
        }
      }, 1000)
    }
    return () => {
      if (pollingIntervalRef.current) {
        clearInterval(pollingIntervalRef.current)
        pollingIntervalRef.current = null
      }
    }
  }, [pollingDeposit])

  useEffect(() => {
    if (pollingSeconds >= 30) {
      setPollingDeposit(false)
    }
  }, [pollingSeconds])

  const totalSteps = STEPS.length

  function handleCopyKey() {
    if (createdKey?.key) {
      copyToClipboard(createdKey.key)
      setKeyCopied(true)
      setTimeout(() => setKeyCopied(false), 3000)
    }
  }

  function renderStep() {
    switch (step) {
      case 0:
        return <WelcomeStep onNext={() => setStep(1)} userEmail={user?.email ?? ""} />
      case 1:
        return (
          <DepositStep
            depositData={depositData}
            pollingDeposit={pollingDeposit}
            pollingSeconds={pollingSeconds}
            depositConfirmed={depositConfirmed}
            onDeposit={() => {
              setPollingDeposit(true)
              setPollingSeconds(0)
            }}
            onSkip={() => setStep(2)}
          />
        )
      case 2:
        return (
          <ApiKeyStep
            label={apiKeyLabel}
            onLabelChange={setApiKeyLabel}
            createdKey={createdKey}
            keyCopied={keyCopied}
            onCopy={handleCopyKey}
            creating={createKeyMutation.isPending}
            error={createKeyError}
            onCreate={() => {
              setCreateKeyError("")
              createKeyMutation.mutate(apiKeyLabel)
            }}
            onNext={() => setStep(3)}
          />
        )
      case 3:
        return (
          <DiscoverStep
            provider={httpbinProvider}
            endpoints={httpbinEndpoints ?? []}
            error={discoverError}
            onNext={() => setStep(4)}
          />
        )
      case 4:
        return (
          <FinishStep
            onFinish={() => completeMutation.mutate()}
            completing={completeMutation.isPending}
            error={completeError}
          />
        )
      default:
        return null
    }
  }

  return (
    <div className="mx-auto max-w-3xl">
      <StepIndicator current={step} total={totalSteps} />

      <Card className="border-primary/10 shadow-sm">
        <CardContent className="p-8">{renderStep()}</CardContent>
      </Card>

      <div className="mt-6 flex items-center justify-between">
        <div>
          {step > 0 && (
            <Button variant="ghost" size="sm" onClick={() => setStep(step - 1)}>
              <ArrowLeft className="mr-1.5 size-4" />
              Back
            </Button>
          )}
        </div>
        <p className="text-xs text-muted-foreground">
          Step {step + 1} of {totalSteps}
        </p>
      </div>
    </div>
  )
}

// ── Step 1: Welcome ──

function WelcomeStep({ onNext, userEmail }: { onNext: () => void; userEmail: string }) {
  return (
    <div className="flex flex-col items-center text-center">
      <div className="mb-6 flex size-16 items-center justify-center rounded-full bg-primary/10">
        <Star className="size-8 text-primary" />
      </div>
      <h2 className="mb-2 text-2xl font-bold tracking-tight">
        Welcome to Castellan
      </h2>
      <p className="mb-2 text-sm text-muted-foreground">
        Hi <span className="font-medium text-foreground">{userEmail}</span>, you&apos;re
        about to experience the easiest way to monetize your APIs.
      </p>
      <p className="mb-8 text-sm text-muted-foreground">
        Castellan handles usage tracking, rate limiting, billing, and Stellar
        settlement — so you can focus on building.
      </p>

      <div className="mb-8 grid w-full gap-4 sm:grid-cols-3">
        {[
          { icon: Wallet, label: "1. Deposit XLM", desc: "Fund your account" },
          { icon: Key, label: "2. Create a key", desc: "Authenticate requests" },
          { icon: Server, label: "3. Publish APIs", desc: "Start earning" },
        ].map((item) => (
          <div
            key={item.label}
            className="flex flex-col items-center rounded-lg border bg-card p-4 text-center"
          >
            <div className="mb-2 flex size-10 items-center justify-center rounded-lg bg-primary/10">
              <item.icon className="size-5 text-primary" />
            </div>
            <span className="text-sm font-medium">{item.label}</span>
            <span className="text-xs text-muted-foreground">{item.desc}</span>
          </div>
        ))}
      </div>

      <Button onClick={onNext}>
        Let&apos;s get started
        <ArrowRight className="ml-1.5 size-4" />
      </Button>
    </div>
  )
}

// ── Step 2: Deposit ──

function DepositStep({
  depositData,
  pollingDeposit,
  pollingSeconds,
  depositConfirmed,
  onDeposit,
  onSkip,
}: {
  depositData: IntentResponse | undefined
  pollingDeposit: boolean
  pollingSeconds: number
  depositConfirmed: boolean
  onDeposit: () => void
  onSkip: () => void
}) {
  return (
    <div className="flex flex-col items-center text-center">
      <div className="mb-6 flex size-16 items-center justify-center rounded-full bg-primary/10">
        <Wallet className="size-8 text-primary" />
      </div>
      <h2 className="mb-2 text-2xl font-bold tracking-tight">
        Deposit XLM
      </h2>
      <p className="mb-8 text-sm text-muted-foreground">
        Add funds to your account to pay for API usage. You can use{" "}
        <strong>Lobstr Wallet</strong> or any Stellar wallet that supports SEP-7
        QR codes.
      </p>

      <div className="mb-6 inline-flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-xs font-medium">
        <span
          className={`size-1.5 rounded-full ${
            getStellarNetworkLabel() === "Mainnet"
              ? "bg-emerald-500"
              : "bg-amber-500"
          }`}
        />
        {getStellarNetworkLabel()}
      </div>

      <div className="mb-8 w-full max-w-md">
        {depositData ? (
          <div className="flex flex-col items-center gap-4 rounded-lg border bg-card p-6">
            {depositData.qr_code ? (
              <img
                src={depositData.qr_code}
                alt="Deposit QR"
                className="size-48 rounded-lg border"
              />
            ) : (
              <div className="flex size-48 items-center justify-center rounded-lg bg-muted">
                <span className="text-xs text-muted-foreground">No QR</span>
              </div>
            )}
            <p className="text-xs text-muted-foreground">
              Scan with Lobstr to auto-fill
            </p>
            <div className="w-full space-y-2 text-left text-sm">
              <div>
                <span className="text-xs text-muted-foreground">Destination</span>
                <p className="font-mono text-xs">{depositData.destination}</p>
              </div>
              <div>
                <span className="text-xs text-muted-foreground">Memo</span>
                <p className="font-mono text-xs">{depositData.memo}</p>
              </div>
              <div>
                <span className="text-xs text-muted-foreground">Min amount</span>
                <p className="font-mono text-xs">
                  {depositData.minimum_amount} {depositData.asset}
                </p>
              </div>
            </div>
            <a
              href={depositData.sep7_uri}
              target="_blank"
              rel="noopener noreferrer"
              className={buttonVariants({ variant: "default", size: "sm" })}
            >
              <Wallet className="mr-1.5 size-4" />
              Open in Lobstr
            </a>
          </div>
        ) : (
          <Skeleton className="h-80 w-full rounded-lg" />
        )}
      </div>

      {depositConfirmed ? (
        <p className="mb-4 text-sm font-medium text-green-600">
          Deposit detected! Your balance has been updated.
        </p>
      ) : pollingDeposit ? (
        <div className="mb-4 flex items-center gap-2 text-sm text-muted-foreground">
          <Loader2 className="size-4 animate-spin" />
          Waiting for deposit ({pollingSeconds}s)…
        </div>
      ) : pollingSeconds >= 30 ? (
        <p className="mb-4 text-sm text-amber-600">
          Deposit not yet detected. You can skip and deposit later.
        </p>
      ) : null}

      <div className="flex gap-3">
        {!depositConfirmed && pollingSeconds === 0 && (
          <Button onClick={onDeposit} variant="outline">
            I&apos;ve deposited
          </Button>
        )}
        {(depositConfirmed || pollingSeconds >= 30) && (
          <Button onClick={onSkip}>
            Next
            <ArrowRight className="ml-1.5 size-4" />
          </Button>
        )}
        {!depositConfirmed && pollingSeconds === 0 && (
          <Button onClick={onSkip} variant="ghost">
            Skip for now
          </Button>
        )}
      </div>
    </div>
  )
}

// ── Step 3: API Key ──

function ApiKeyStep({
  label,
  onLabelChange,
  createdKey,
  keyCopied,
  onCopy,
  creating,
  error,
  onCreate,
  onNext,
}: {
  label: string
  onLabelChange: (v: string) => void
  createdKey: CreateApiKeyResponse | null
  keyCopied: boolean
  onCopy: () => void
  creating: boolean
  error: string
  onCreate: () => void
  onNext: () => void
}) {
  return (
    <div className="flex flex-col items-center text-center">
      <div className="mb-6 flex size-16 items-center justify-center rounded-full bg-primary/10">
        <Key className="size-8 text-primary" />
      </div>
      <h2 className="mb-2 text-2xl font-bold tracking-tight">
        Create an API Key
      </h2>
      <p className="mb-8 text-sm text-muted-foreground">
        Generate an API key to authenticate your requests. You&apos;ll use this
        in the <code className="rounded bg-muted px-1">X-API-Key</code> header.
      </p>

      {!createdKey ? (
        <div className="mb-8 w-full max-w-sm space-y-4">
          <div className="text-left">
            <Label htmlFor="key-label">Key label</Label>
            <Input
              id="key-label"
              value={label}
              onChange={(e) => onLabelChange(e.target.value)}
              placeholder="My First Key"
              className="mt-1.5"
            />
          </div>
          {error && (
            <p className="text-sm text-red-500">{error}</p>
          )}
          <Button
            onClick={onCreate}
            disabled={creating || !label.trim()}
            className="w-full"
          >
            {creating ? (
              <>
                <Loader2 className="mr-1.5 size-4 animate-spin" />
                Generating…
              </>
            ) : (
              "Generate API Key"
            )}
          </Button>
        </div>
      ) : (
        <div className="mb-8 w-full max-w-lg space-y-4">
          <div className="rounded-lg border border-amber-200 bg-amber-50 p-4 text-left text-sm text-amber-800 dark:border-amber-800 dark:bg-amber-950 dark:text-amber-200">
            <Shield className="mb-1 inline-block size-4" />
            {" "}
            Copy this key now. You won&apos;t be able to see it again.
          </div>
          <div className="flex gap-2">
            <code className="flex-1 overflow-hidden text-ellipsis rounded-lg border bg-muted px-3 py-2.5 font-mono text-xs whitespace-nowrap">
              {createdKey.key}
            </code>
            <Button variant="outline" size="icon" onClick={onCopy}>
              {keyCopied ? (
                <Check className="size-4 text-green-500" />
              ) : (
                <Copy className="size-4" />
              )}
            </Button>
          </div>
          <Button onClick={onNext} className="w-full">
            I&apos;ve saved my key
            <ArrowRight className="ml-1.5 size-4" />
          </Button>
        </div>
      )}
    </div>
  )
}

// ── Step 4: Discover httpbin ──

function DiscoverStep({
  provider,
  endpoints,
  error,
  onNext,
}: {
  provider: Provider | undefined
  endpoints: Endpoint[]
  error: boolean
  onNext: () => void
}) {
  const postEndpoint = endpoints.find((e) => e.method === "POST")
  const { data: config } = useQuery<ApiConfigResponse>({
    queryKey: ["api-config"],
    queryFn: getApiConfig,
    staleTime: Infinity,
  })
  const apiBase = config?.api_base_url || "http://localhost:8080"
  const [curlCopied, setCurlCopied] = useState(false)

  const handleCopyCurl = () => {
    if (!provider || !postEndpoint) return
    const cmd =
      `curl -X POST "${apiBase}/api/gateway/${provider.name}${postEndpoint.route}" \\` +
      `\n  -H "Content-Type: application/json" \\` +
      `\n  -H "Authorization: Bearer <your-api-key>" \\` +
      `\n  -d '{"hello": "world"}'`
    copyToClipboard(cmd)
    setCurlCopied(true)
    setTimeout(() => setCurlCopied(false), 2000)
  }

  return (
    <div className="flex flex-col items-center text-center">
      <div className="mb-6 flex size-16 items-center justify-center rounded-full bg-primary/10">
        <Search className="size-8 text-primary" />
      </div>
      <h2 className="mb-2 text-2xl font-bold tracking-tight">
        Test Your First API
      </h2>
      <p className="mb-8 text-sm text-muted-foreground">
        We&apos;ve pre-configured a demo{" "}
        <strong>httpbin.org</strong> provider for you.
      </p>

      {provider ? (
        <div className="mb-8 w-full max-w-lg space-y-6 text-left">
          <div className="rounded-lg border bg-card p-4">
            <div className="flex items-center gap-2">
              <Server className="size-4 text-primary" />
              <span className="font-medium">{provider.name}</span>
            </div>
            <p className="mt-1 text-sm text-muted-foreground">
              Base URL:{" "}
              <code className="rounded bg-muted px-1 font-mono text-xs">
                {provider.base_url}
              </code>
            </p>
          </div>

          {endpoints.length > 0 && (
            <div className="rounded-lg border bg-card p-4">
              <h4 className="mb-3 text-sm font-medium">Available endpoints</h4>
              <div className="space-y-2">
                {endpoints.slice(0, 3).map((ep) => (
                  <div
                    key={ep.id}
                    className="flex items-center gap-2 rounded-md bg-muted/50 p-2 text-sm"
                  >
                    <MethodBadge method={ep.method} />
                    <code className="font-mono text-xs">{ep.route}</code>
                    <span className="ml-auto text-xs text-muted-foreground">
                      {ep.price_amount} {ep.currency}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          )}

          {postEndpoint && (
            <div className="rounded-lg border bg-card p-4">
              <div className="mb-2 flex items-center justify-between">
                <h4 className="flex items-center gap-2 text-sm font-medium">
                  <Terminal className="size-4" />
                  Try it out
                </h4>
                <button
                  type="button"
                  onClick={handleCopyCurl}
                  className="inline-flex items-center justify-center rounded-md p-1.5 text-muted-foreground hover:bg-muted hover:text-foreground"
                >
                  {curlCopied ? (
                    <Check className="size-4 text-green-500" />
                  ) : (
                    <Copy className="size-4" />
                  )}
                </button>
              </div>
              <p className="mb-2 text-xs text-muted-foreground">
                Replace {"<your-api-key>"} with the key from the previous step.
              </p>
              <div className="overflow-x-auto rounded-md bg-muted p-3">
                <pre className="font-mono text-xs leading-relaxed">
                  {`curl -X POST "`}{apiBase}/api/gateway/{provider.name}{postEndpoint.route}{`" \\`}
                  {`\n  -H "Content-Type: application/json" \\`}
                  {`\n  -H "Authorization: Bearer <your-api-key>" \\`}
                  {`\n  -d '{"hello": "world"}'`}
                </pre>
              </div>
            </div>
          )}

          <div className="rounded-lg border border-amber-200 bg-amber-50 p-3 text-left text-sm text-amber-800 dark:border-amber-800 dark:bg-amber-950 dark:text-amber-200">
            Make sure to replace{" "}
            <code className="rounded bg-amber-100 px-1 font-mono text-xs dark:bg-amber-900 dark:text-amber-300">{"<your-api-key>"}</code>{" "}
            with the actual API key you generated.
          </div>
        </div>
      ) : error ? (
        <div className="mb-8 w-full max-w-lg rounded-lg border bg-card p-6 text-center">
          <p className="text-sm text-muted-foreground">
            The demo httpbin provider isn&apos;t available. You can skip this step.
          </p>
        </div>
      ) : (
        <Skeleton className="mb-8 h-48 w-full max-w-lg rounded-lg" />
      )}

      <Button onClick={onNext}>
        I ran the curl command
        <ArrowRight className="ml-1.5 size-4" />
      </Button>
    </div>
  )
}

// ── Step 5: Finish ──

function FinishStep({
  onFinish,
  completing,
  error,
}: {
  onFinish: () => void
  completing: boolean
  error: string
}) {
  return (
    <div className="flex flex-col items-center text-center">
      <div className="mb-6 flex size-16 items-center justify-center rounded-full bg-green-100">
        <PartyPopper className="size-8 text-green-600" />
      </div>
      <h2 className="mb-2 text-2xl font-bold tracking-tight">
        You&apos;re All Set!
      </h2>
      <p className="mb-2 text-sm text-muted-foreground">
        Here&apos;s what you&apos;ve done:
      </p>

      <div className="mb-8 w-full max-w-md space-y-3 text-left">
        {[
          { icon: Wallet, text: "Learned how to deposit XLM" },
          { icon: Key, text: "Created your first API key" },
          { icon: Terminal, text: "Tested an API call against httpbin" },
        ].map((item) => (
          <div key={item.text} className="flex items-center gap-3 rounded-lg bg-muted/50 p-3">
            <div className="flex size-8 items-center justify-center rounded-full bg-primary/10">
              <item.icon className="size-4 text-primary" />
            </div>
            <span className="text-sm">{item.text}</span>
          </div>
        ))}
      </div>

      <p className="mb-8 text-sm text-muted-foreground">
        Congratulations! You&apos;re ready to start building with Castellan.
      </p>

      {error && (
        <p className="mb-4 text-sm text-red-500">{error}</p>
      )}
      <Button onClick={onFinish} disabled={completing} size="lg">
        {completing ? (
          <>
            <Loader2 className="mr-1.5 size-4 animate-spin" />
            Finishing…
          </>
        ) : (
          <>
            Go to Dashboard
            <ArrowRight className="ml-1.5 size-4" />
          </>
        )}
      </Button>
    </div>
  )
}
