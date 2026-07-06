"use client"

import { useState, useMemo } from "react"
import { Search, Compass, TrendingUp, ArrowRight, AlertCircle } from "lucide-react"
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { PageHeader } from "@/components/PageHeader"
import { StatCard } from "@/components/StatCard"
import { EmptyState } from "@/components/EmptyState"
import { StatusBadge } from "@/components/StatusBadge"
import { CurrencyDisplay } from "@/components/CurrencyDisplay"
import { StellarLogo } from "@/components/StellarLogo"
import { MOCK_DISCOVERABLE_APIS } from "@/lib/mock-data"
import type { DiscoverableAPI } from "@/lib/types"

export default function DiscoverPage() {
  const [searchQuery, setSearchQuery] = useState("")
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(false)

  const activeAPIs = useMemo(
    () => MOCK_DISCOVERABLE_APIS.filter((a) => a.status === "active"),
    []
  )

  const filtered = useMemo(() => {
    if (!searchQuery) return activeAPIs
    const q = searchQuery.toLowerCase()
    return activeAPIs.filter(
      (a) =>
        a.name.toLowerCase().includes(q) ||
        a.description.toLowerCase().includes(q) ||
        a.endpoint.toLowerCase().includes(q) ||
        a.category.toLowerCase().includes(q)
    )
  }, [activeAPIs, searchQuery])

  const priceRange = useMemo(() => {
    const prices = activeAPIs.map((a) => a.price)
    const min = Math.min(...prices)
    const max = Math.max(...prices)
    return `${min.toFixed(3)} – ${max.toFixed(2)}`
  }, [activeAPIs])

  const topCategory = useMemo(() => {
    const counts: Record<string, number> = {}
    for (const a of activeAPIs) {
      counts[a.category] = (counts[a.category] || 0) + 1
    }
    return Object.entries(counts).sort((a, b) => b[1] - a[1])[0]?.[0] ?? ""
  }, [activeAPIs])

  if (error) {
    return (
      <div className="flex flex-col items-center gap-4 py-20">
        <AlertCircle className="h-12 w-12 text-destructive" />
        <h2 className="text-xl font-semibold">Failed to load APIs</h2>
        <Button onClick={() => setError(false)}>Retry</Button>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="Discover APIs"
        description="Browse available APIs on the Castellan network. Subscribe to any API to start making requests."
      />

      {loading ? (
        <div className="grid gap-4 sm:grid-cols-3">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-24" />
          ))}
        </div>
      ) : (
        <div className="grid gap-4 sm:grid-cols-3">
          <StatCard
            title="Available APIs"
            value={String(activeAPIs.length)}
            icon={Compass}
          />
          <StatCard
            title="Price Range"
            value={<span className="inline-flex items-center gap-1.5 text-2xl font-semibold tracking-tight"><StellarLogo className="size-5" />{priceRange}</span>}
            icon={TrendingUp}
          />
          <StatCard
            title="Most Popular"
            value={topCategory}
            icon={TrendingUp}
          />
        </div>
      )}

      <div className="relative max-w-md">
        <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        <Input
          placeholder="Search APIs by name, description, endpoint..."
          className="pl-9"
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
        />
      </div>

      {filtered.length === 0 ? (
        <EmptyState
          icon={searchQuery ? Search : Compass}
          title={searchQuery ? "No APIs match your search" : "No APIs available yet"}
          description={
            searchQuery
              ? "Try a different search term."
              : "Check back later for new APIs."
          }
        />
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {filtered.map((api) => (
            <Card key={api.id} className="flex flex-col">
              <CardHeader>
                <div className="flex items-start justify-between gap-2">
                  <CardTitle className="text-base">{api.name}</CardTitle>
                  <StatusBadge status={api.status} />
                </div>
                <Badge variant="secondary" className="w-fit">
                  {api.category}
                </Badge>
              </CardHeader>
              <CardContent className="flex-1 space-y-2">
                <p className="text-sm text-muted-foreground">{api.description}</p>
                <p className="font-mono text-xs text-muted-foreground">{api.endpoint}</p>
                <p className="text-sm font-medium">
                  <CurrencyDisplay amount={api.price} /> / request
                </p>
              </CardContent>
              <CardFooter className="flex gap-2">
                <Button size="sm">
                  Subscribe <ArrowRight className="ml-1 h-3.5 w-3.5" />
                </Button>
                <Button variant="outline" size="sm">
                  View Docs
                </Button>
              </CardFooter>
            </Card>
          ))}
        </div>
      )}
    </div>
  )
}
