"use client"

import { useState, useEffect } from "react"
import { Search } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"

export function SearchCommand() {
  const [open, setOpen] = useState(false)

  useEffect(() => {
    const down = (e: KeyboardEvent) => {
      if (e.key === "k" && (e.metaKey || e.ctrlKey)) {
        e.preventDefault()
        setOpen((prev) => !prev)
      }
    }
    document.addEventListener("keydown", down)
    return () => document.removeEventListener("keydown", down)
  }, [])

  return (
    <>
      <Button
        variant="outline"
        className="h-8 w-full max-w-xs justify-between gap-2 text-xs text-muted-foreground"
        onClick={() => setOpen(true)}
      >
        <div className="flex items-center gap-1.5">
          <Search className="size-3.5" />
          <span>Search...</span>
        </div>
        <kbd className="pointer-events-none inline-flex h-5 items-center gap-0.5 rounded border bg-muted px-1 font-mono text-[10px] font-medium text-muted-foreground">
          <span className="text-[9px]">⌘</span>K
        </kbd>
      </Button>
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="sm:max-w-lg" showCloseButton={false}>
          <DialogHeader>
            <DialogTitle className="sr-only">Search</DialogTitle>
          </DialogHeader>
          <div className="flex items-center gap-2 border-b pb-3">
            <Search className="size-4 text-muted-foreground" />
            <Input
              placeholder="Search APIs, endpoints, providers..."
              className="border-none p-0 text-sm shadow-none focus-visible:ring-0"
              autoFocus
            />
          </div>
          <div className="flex flex-col items-center justify-center gap-1 py-8 text-center">
            <p className="text-xs text-muted-foreground">Type to search across the dashboard</p>
          </div>
        </DialogContent>
      </Dialog>
    </>
  )
}
