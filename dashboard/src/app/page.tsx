"use client"

import Link from "next/link"
import { ArrowRight, Shield, Zap, Globe, Github, Menu, X, Code2 } from "lucide-react"
import { Button } from "@/components/ui/button"
import { useState } from "react"

export default function LandingPage() {
  const [mobileOpen, setMobileOpen] = useState(false)

  return (
    <div className="flex min-h-screen flex-col bg-background">
      <header className="fixed top-0 z-50 w-full border-b bg-background/80 backdrop-blur-sm">
        <div className="mx-auto flex h-16 max-w-7xl items-center justify-between px-4 lg:px-8">
          <Link href="/" className="flex items-center gap-2">
            <div className="flex h-8 w-8 items-center justify-center rounded-md bg-primary">
              <span className="text-sm font-bold text-primary-foreground">C</span>
            </div>
            <span className="text-lg font-semibold tracking-tight">Castellan</span>
          </Link>

          <nav className="hidden items-center gap-8 md:flex">
            <Link href="#how-it-works" className="text-sm text-muted-foreground hover:text-foreground transition-colors">How it works</Link>
            <Link href="#for-providers" className="text-sm text-muted-foreground hover:text-foreground transition-colors">For Providers</Link>
            <Link href="#for-consumers" className="text-sm text-muted-foreground hover:text-foreground transition-colors">For Consumers</Link>
            <Link href="#open-source" className="text-sm text-muted-foreground hover:text-foreground transition-colors">Open Source</Link>
          </nav>

          <div className="hidden items-center gap-3 md:flex">
            <Button variant="ghost" size="sm" asChild>
              <Link href="/login">Sign In</Link>
            </Button>
            <Button size="sm" asChild>
              <Link href="/signup">Start Building</Link>
            </Button>
          </div>

          <Button variant="ghost" size="icon" className="md:hidden" onClick={() => setMobileOpen(!mobileOpen)}>
            {mobileOpen ? <X className="h-5 w-5" /> : <Menu className="h-5 w-5" />}
          </Button>
        </div>

        {mobileOpen && (
          <div className="border-b bg-background px-4 pb-4 md:hidden">
            <nav className="flex flex-col gap-3 pt-4">
              <Link href="#how-it-works" className="text-sm text-muted-foreground" onClick={() => setMobileOpen(false)}>How it works</Link>
              <Link href="#for-providers" className="text-sm text-muted-foreground" onClick={() => setMobileOpen(false)}>For Providers</Link>
              <Link href="#for-consumers" className="text-sm text-muted-foreground" onClick={() => setMobileOpen(false)}>For Consumers</Link>
              <Link href="#open-source" className="text-sm text-muted-foreground" onClick={() => setMobileOpen(false)}>Open Source</Link>
              <div className="flex gap-2 pt-2">
                <Button variant="outline" size="sm" className="flex-1" asChild>
                  <Link href="/login">Sign In</Link>
                </Button>
                <Button size="sm" className="flex-1" asChild>
                  <Link href="/signup">Start Building</Link>
                </Button>
              </div>
            </nav>
          </div>
        )}
      </header>

      <section className="relative overflow-hidden pt-32 pb-20 lg:pt-40 lg:pb-28">
        <div className="absolute inset-0 bg-gradient-to-b from-primary/5 via-background to-background" />
        <div className="mx-auto max-w-7xl px-4 text-center lg:px-8 relative">
          <div className="mb-6 inline-flex items-center gap-2 rounded-full border bg-muted/50 px-4 py-1.5 text-xs text-muted-foreground">
            <Zap className="h-3 w-3 text-primary" />
            Stellar-powered API monetization
          </div>
          <h1 className="mx-auto max-w-4xl text-4xl font-bold tracking-tight sm:text-5xl lg:text-6xl">
            Usage-based API
            <br />
            <span className="text-primary">monetization infrastructure</span>
          </h1>
          <p className="mx-auto mt-6 max-w-2xl text-base text-muted-foreground lg:text-lg">
            Wrap your API, set per-request pricing, and get paid instantly via Stellar.
            No subscription models. No Stripe overhead. Just transparent billing for every call.
          </p>
          <div className="mt-10 flex items-center justify-center gap-4">
            <Button size="lg" asChild>
              <Link href="/signup">
                Start Building <ArrowRight className="ml-2 h-4 w-4" />
              </Link>
            </Button>
            <Button variant="outline" size="lg" asChild>
              <Link href="#how-it-works">Learn More</Link>
            </Button>
          </div>
          <div className="mt-16 flex items-center justify-center gap-8 text-xs text-muted-foreground">
            <span className="flex items-center gap-1.5"><Shield className="h-3.5 w-3.5" /> Stellar settlement</span>
            <span className="flex items-center gap-1.5"><Zap className="h-3.5 w-3.5" /> Real-time billing</span>
            <span className="flex items-center gap-1.5"><Globe className="h-3.5 w-3.5" /> Open source</span>
          </div>
        </div>
      </section>

      <section id="how-it-works" className="border-t py-20 lg:py-28">
        <div className="mx-auto max-w-7xl px-4 lg:px-8">
          <div className="text-center">
            <h2 className="text-3xl font-bold tracking-tight">How It Works</h2>
            <p className="mt-3 text-muted-foreground">Two flows, one platform. Choose your path.</p>
          </div>
          <div className="mt-16 grid gap-8 lg:grid-cols-2">
            <div className="rounded-xl border bg-card p-8">
              <div className="mb-4 inline-flex items-center gap-2 rounded-full bg-primary/10 px-3 py-1 text-xs font-medium text-primary">
                Provider Flow
              </div>
              <div className="space-y-8">
                {[
                  { step: "01", title: "Wrap Your API", desc: "Register your API with Castellan. Point to your base URL and define your endpoints." },
                  { step: "02", title: "Set Pricing", desc: "Set per-request prices in XLM for each endpoint. Configure rate limits per route." },
                  { step: "03", title: "Get Paid", desc: "Earn XLM for every request. Settlements are sent directly to your Stellar wallet." },
                ].map((item) => (
                  <div key={item.step} className="flex gap-4">
                    <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-sm font-bold text-primary">{item.step}</span>
                    <div>
                      <h3 className="font-semibold">{item.title}</h3>
                      <p className="text-sm text-muted-foreground">{item.desc}</p>
                    </div>
                  </div>
                ))}
              </div>
            </div>
            <div className="rounded-xl border bg-card p-8">
              <div className="mb-4 inline-flex items-center gap-2 rounded-full bg-green/10 px-3 py-1 text-xs font-medium text-green">
                Consumer Flow
              </div>
              <div className="space-y-8">
                {[
                  { step: "01", title: "Get a Key", desc: "Sign up, choose a role, and generate your API key. No credit card required." },
                  { step: "02", title: "Deposit XLM", desc: "Send XLM to your Castellan wallet via Stellar. Funds are credited in seconds." },
                  { step: "03", title: "Use APIs", desc: "Call any API on the network. Pay only for what you use, per request." },
                ].map((item) => (
                  <div key={item.step} className="flex gap-4">
                    <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-green/10 text-sm font-bold text-green">{item.step}</span>
                    <div>
                      <h3 className="font-semibold">{item.title}</h3>
                      <p className="text-sm text-muted-foreground">{item.desc}</p>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>
      </section>

      <section id="for-providers" className="border-t py-20 lg:py-28">
        <div className="mx-auto max-w-7xl px-4 lg:px-8">
          <div className="grid gap-12 lg:grid-cols-2 lg:gap-16 items-center">
            <div>
              <h2 className="text-3xl font-bold tracking-tight">Built for API Providers</h2>
              <p className="mt-4 text-muted-foreground leading-relaxed">
                Stop wrestling with subscription billing. Castellan handles usage-based pricing,
                token-gated access, and blockchain settlement so you can focus on building.
              </p>
              <ul className="mt-8 space-y-4">
                {[
                  { title: "Per-request billing", desc: "Set granular prices per endpoint. No subscription tiers needed." },
                  { title: "Stellar settlement", desc: "Instant payouts directly to your Stellar wallet. No bank delays." },
                  { title: "Rate limiting", desc: "Control traffic with per-endpoint rate limits and burst configurations." },
                  { title: "Real-time analytics", desc: "Monitor usage, revenue, and latency in real time." },
                ].map((item) => (
                  <li key={item.title} className="flex gap-3">
                    <div className="mt-1 h-5 w-5 rounded-full bg-primary/10 flex items-center justify-center shrink-0">
                      <div className="h-2 w-2 rounded-full bg-primary" />
                    </div>
                    <div>
                      <p className="text-sm font-medium">{item.title}</p>
                      <p className="text-sm text-muted-foreground">{item.desc}</p>
                    </div>
                  </li>
                ))}
              </ul>
            </div>
            <div className="rounded-xl border bg-card p-6">
              <div className="flex items-center gap-2 mb-4">
                <Code2 className="h-4 w-4 text-primary" />
                <span className="text-sm font-medium">Example: Register your API</span>
              </div>
              <pre className="overflow-x-auto text-xs text-muted-foreground">
                <code className="block whitespace-pre">{`curl -X POST https://api.castellan.io/v1/providers \\
  -H "Authorization: Bearer fg_your_key_xxxxxxxx" \\
  -H "Content-Type: application/json" \\
  -d '{
    "name": "Weather API",
    "base_url": "https://api.weather.com",
    "endpoints": [
      {
        "route": "/current",
        "method": "GET",
        "price": "0.0001",
        "rate_limit": 100
      }
    ]
  }'`}</code>
              </pre>
            </div>
          </div>
        </div>
      </section>

      <section id="for-consumers" className="border-t bg-muted/30 py-20 lg:py-28">
        <div className="mx-auto max-w-7xl px-4 lg:px-8">
          <div className="grid gap-12 lg:grid-cols-2 lg:gap-16 items-center">
            <div className="order-2 lg:order-1 rounded-xl border bg-card p-6">
              <div className="flex items-center gap-2 mb-4">
                <Code2 className="h-4 w-4 text-primary" />
                <span className="text-sm font-medium">Example: Call an API</span>
              </div>
              <pre className="overflow-x-auto text-xs text-muted-foreground">
                <code className="block whitespace-pre">{`curl https://api.castellan.io/proxy/weather/current \\
  -H "Authorization: Bearer fg_your_key_xxxxxxxx"

# Response
{
  "temperature": 22,
  "conditions": "partly_cloudy",
  "cost": "0.0001 XLM",
  "balance_remaining": "1249.99 XLM"
}`}</code>
              </pre>
            </div>
            <div className="order-1 lg:order-2">
              <h2 className="text-3xl font-bold tracking-tight">Built for API Consumers</h2>
              <p className="mt-4 text-muted-foreground leading-relaxed">
                No more monthly subscriptions for APIs you use twice a year.
                Prepay with XLM and only spend when you make a request.
              </p>
              <ul className="mt-8 space-y-4">
                {[
                  { title: "Pay per use", desc: "Only pay for actual API calls. No monthly commitments." },
                  { title: "Prepaid via Stellar", desc: "Deposit XLM and track your spending in real time." },
                  { title: "Transparent billing", desc: "Every request shows cost and remaining balance in the response headers." },
                  { title: "Discover APIs", desc: "Browse available providers and endpoints from the dashboard." },
                ].map((item) => (
                  <li key={item.title} className="flex gap-3">
                    <div className="mt-1 h-5 w-5 rounded-full bg-green/10 flex items-center justify-center shrink-0">
                      <div className="h-2 w-2 rounded-full bg-green" />
                    </div>
                    <div>
                      <p className="text-sm font-medium">{item.title}</p>
                      <p className="text-sm text-muted-foreground">{item.desc}</p>
                    </div>
                  </li>
                ))}
              </ul>
            </div>
          </div>
        </div>
      </section>

      <section className="border-t py-20 lg:py-28">
        <div className="mx-auto max-w-7xl px-4 text-center lg:px-8">
          <h2 className="text-3xl font-bold tracking-tight">Architecture</h2>
          <p className="mt-3 text-muted-foreground">How requests flow through the Castellan gateway.</p>
          <div className="mt-12 rounded-xl border bg-card p-8 lg:p-12">
            <div className="flex flex-col items-center gap-4 sm:flex-row sm:justify-center">
              <div className="flex flex-col items-center gap-2 rounded-lg border bg-background px-6 py-4">
                <Code2 className="h-5 w-5 text-muted-foreground" />
                <span className="text-xs font-medium">Your Client</span>
              </div>
              <ArrowRight className="h-5 w-5 text-muted-foreground" />
              <div className="flex flex-col items-center gap-2 rounded-lg border border-primary/30 bg-primary/5 px-6 py-4">
                <Shield className="h-5 w-5 text-primary" />
                <span className="text-xs font-medium">Castellan Gateway</span>
                <span className="text-[10px] text-muted-foreground">Auth → Balance → Proxy</span>
              </div>
              <ArrowRight className="h-5 w-5 text-muted-foreground" />
              <div className="flex flex-col items-center gap-2 rounded-lg border bg-background px-6 py-4">
                <Globe className="h-5 w-5 text-muted-foreground" />
                <span className="text-xs font-medium">Your API</span>
              </div>
            </div>
            <p className="mt-6 text-xs text-muted-foreground">
              Castellan authenticates, checks balance, deducts XLM, and proxies the request — all in one pipeline.
            </p>
          </div>
        </div>
      </section>

      <section id="open-source" className="border-t bg-background py-20 lg:py-28">
        <div className="mx-auto max-w-7xl px-4 text-center lg:px-8">
          <div className="mx-auto max-w-2xl">
            <div className="mb-6 inline-flex items-center gap-2 rounded-full border bg-card px-4 py-1.5 text-xs text-muted-foreground">
              <Github className="h-3.5 w-3.5" /> Apache 2.0 License
            </div>
            <h2 className="text-3xl font-bold tracking-tight">Open Source</h2>
            <p className="mt-4 text-muted-foreground">
              Castellan is Apache 2.0 licensed. Self-host on your own infrastructure or use our managed cloud.
              Contributions, issues, and feature requests are welcome.
            </p>
            <div className="mt-8 flex items-center justify-center gap-4">
              <Button variant="outline" asChild>
                <Link href="https://github.com/castellan" target="_blank">
                  <Github className="mr-2 h-4 w-4" /> View on GitHub
                </Link>
              </Button>
              <Button asChild>
                <Link href="/signup">Try Castellan Cloud</Link>
              </Button>
            </div>
          </div>
        </div>
      </section>

      <footer className="border-t py-12">
        <div className="mx-auto max-w-7xl px-4 lg:px-8">
          <div className="grid gap-8 sm:grid-cols-2 lg:grid-cols-4">
            <div>
              <div className="flex items-center gap-2 mb-4">
                <div className="flex h-6 w-6 items-center justify-center rounded bg-primary">
                  <span className="text-[10px] font-bold text-primary-foreground">C</span>
                </div>
                <span className="text-sm font-semibold">Castellan</span>
              </div>
              <p className="text-xs text-muted-foreground">Usage-based API monetization on Stellar.</p>
            </div>
            <div>
              <h4 className="mb-3 text-xs font-semibold uppercase tracking-wider text-muted-foreground">Product</h4>
              <ul className="space-y-2 text-sm text-muted-foreground">
                <li><Link href="#how-it-works" className="hover:text-foreground transition-colors">How it works</Link></li>
                <li><Link href="#for-providers" className="hover:text-foreground transition-colors">For Providers</Link></li>
                <li><Link href="#for-consumers" className="hover:text-foreground transition-colors">For Consumers</Link></li>
              </ul>
            </div>
            <div>
              <h4 className="mb-3 text-xs font-semibold uppercase tracking-wider text-muted-foreground">Resources</h4>
              <ul className="space-y-2 text-sm text-muted-foreground">
                <li><Link href="/docs" className="hover:text-foreground transition-colors">API Docs</Link></li>
                <li><Link href="https://github.com/castellan" className="hover:text-foreground transition-colors">GitHub</Link></li>
                <li><span className="cursor-default">Status</span></li>
              </ul>
            </div>
            <div>
              <h4 className="mb-3 text-xs font-semibold uppercase tracking-wider text-muted-foreground">Legal</h4>
              <ul className="space-y-2 text-sm text-muted-foreground">
                <li><span className="cursor-default">Terms</span></li>
                <li><span className="cursor-default">Privacy</span></li>
              </ul>
            </div>
          </div>
          <div className="mt-10 border-t pt-6 text-center text-xs text-muted-foreground">
            &copy; {new Date().getFullYear()} Castellan. Apache 2.0 License.
          </div>
        </div>
      </footer>
    </div>
  )
}
