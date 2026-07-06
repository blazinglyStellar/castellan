"use client"

import { useState } from "react"
import { Search, Compass, Code2, ExternalLink, ChevronDown, ChevronUp, AlertCircle } from "lucide-react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { PageHeader } from "@/components/PageHeader"
import { EmptyState } from "@/components/EmptyState"
import { StatusBadge } from "@/components/StatusBadge"
import { CopyButton } from "@/components/CopyButton"
import { MOCK_PROVIDERS } from "@/lib/mock-data"
import { cn } from "@/lib/utils"
import type { Provider } from "@/lib/mock-data"

const activeProviders = MOCK_PROVIDERS.filter((p) => p.status === "active")

export default function DiscoverPage() {
  const [searchQuery, setSearchQuery] = useState("")
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(false)
  const [expanded, setExpanded] = useState<Set<string>>(new Set())
  const [isEmpty, setIsEmpty] = useState(false)

  const filtered = activeProviders.filter((p) => {
    if (!searchQuery) return true
    const q = searchQuery.toLowerCase()
    return (
      p.name.toLowerCase().includes(q) ||
      p.baseUrl.toLowerCase().includes(q) ||
      p.endpoints.some((e) => e.route.toLowerCase().includes(q))
    )
  })

  const toggleExpanded = (id: string) => {
    setExpanded((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  if (error) {
    return (
      <div className="flex flex-col items-center gap-4 py-20">
        <AlertCircle className="h-12 w-12 text-destructive" />
        <h2 className="text-xl font-semibold">Failed to load providers</h2>
        <Button onClick={() => setError(false)}>Retry</Button>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="Discover APIs"
        description="Browse available APIs on the Castellan network."
      />

      <div className="relative max-w-md">
        <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        <Input
          placeholder="Search providers, endpoints..."
          className="pl-9"
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
        />
      </div>

      {isEmpty ? (
        <EmptyState
          title="No APIs available yet"
          description="Check back later for new providers."
          icon={<Compass className="h-12 w-12" />}
        />
      ) : loading ? (
        <div className="space-y-4">
          {[1, 2, 3].map((i) => (
            <Skeleton key={i} className="h-40 w-full" />
          ))}
        </div>
      ) : filtered.length === 0 ? (
        <EmptyState
          title="No providers match your search"
          description="Try a different search term."
          icon={<Search className="h-12 w-12" />}
        />
      ) : (
        <div className="space-y-4">
          {filtered.map((provider) => {
            const activeEndpoints = provider.endpoints.filter((e) => e.status === "active")
            const isExpanded = expanded.has(provider.id)
            return (
              <Card key={provider.id} className="overflow-hidden">
                <div
                  className="cursor-pointer"
                  onClick={() => toggleExpanded(provider.id)}
                >
                  <CardHeader className="pb-3">
                    <div className="flex items-start justify-between">
                      <div>
                        <div className="flex items-center gap-2">
                          <CardTitle className="text-base">{provider.name}</CardTitle>
                          <StatusBadge status={provider.status} />
                          <Badge variant="secondary" className="text-xs">
                            {activeEndpoints.length} endpoint{activeEndpoints.length !== 1 ? "s" : ""}
                          </Badge>
                        </div>
                        <p className="mt-1 font-mono text-xs text-muted-foreground">{provider.baseUrl}</p>
                      </div>
                      <div className="flex items-center gap-1">
                        <Button variant="ghost" size="sm" className="h-8 text-xs" onClick={(e) => { e.stopPropagation(); toggleExpanded(provider.id) }}>
                          {isExpanded ? (
                            <><ChevronUp className="mr-1 h-3.5 w-3.5" /> Less</>
                          ) : (
                            <><ChevronDown className="mr-1 h-3.5 w-3.5" /> Details</>
                          )}
                        </Button>
                      </div>
                    </div>
                  </CardHeader>
                </div>

                {isExpanded && (
                  <CardContent className="border-t pt-4">
                    <div className="space-y-3">
                      {activeEndpoints.map((ep) => (
                        <div key={ep.id} className="flex items-center justify-between rounded-lg bg-muted/50 p-3">
                          <div className="flex items-center gap-3">
                            <Badge
                              variant={
                                ep.method === "GET" ? "default" :
                                ep.method === "POST" ? "secondary" :
                                ep.method === "DELETE" ? "destructive" : "outline"
                              }
                              className="font-mono text-[10px] uppercase"
                            >
                              {ep.method}
                            </Badge>
                            <span className="font-mono text-xs">{ep.route}</span>
                            <span className="text-xs text-muted-foreground">{ep.price} XLM</span>
                          </div>
                          <div className="flex items-center gap-2">
                            <span className="text-[11px] text-muted-foreground">{ep.rateLimit}/min</span>
                          </div>
                        </div>
                      ))}
                    </div>

                    <div className="mt-4 flex items-center gap-2">
                      <Button variant="outline" size="sm" className="h-8 text-xs">
                        <Code2 className="mr-1.5 h-3.5 w-3.5" /> Copy curl
                      </Button>
                      <Button variant="ghost" size="sm" className="h-8 text-xs">
                        <ExternalLink className="mr-1.5 h-3.5 w-3.5" /> View Docs
                      </Button>
                    </div>
                  </CardContent>
                )}
              </Card>
            )
          })}
        </div>
      )}
    </div>
  )
}
