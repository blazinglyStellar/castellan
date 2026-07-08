"use client";

import { Code } from "lucide-react";

import { EmptyState } from "@/components/ui/empty-state";

export default function MyApisPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">My APIs</h1>
        <p className="text-sm text-muted-foreground">
          Publish and manage your API catalog.
        </p>
      </div>

      <EmptyState
        icon={Code}
        title="Coming soon"
        description="You'll be able to publish and manage your API catalog here."
      />
    </div>
  );
}
