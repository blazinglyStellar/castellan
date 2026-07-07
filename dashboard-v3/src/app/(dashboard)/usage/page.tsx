"use client";

import { useState } from "react";

import { useAccount } from "@/lib/auth/account-context";
import { FilterBar } from "@/components/usage/filter-bar";
import { UsageTable } from "@/components/usage/usage-table";

export default function UsagePage() {
  const { user, isLoading: isAccountLoading } = useAccount();

  const [role, setRole] = useState<"provider" | "consumer">("provider");
  const [startDate, setStartDate] = useState("");
  const [endDate, setEndDate] = useState("");
  const [statusCode, setStatusCode] = useState(" ");

  if (isAccountLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-muted border-t-primary" />
      </div>
    );
  }

  const resolvedRole = role ?? user?.role ?? "consumer";

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Usage</h1>
        <p className="text-sm text-muted-foreground">
          View and filter API usage events.
        </p>
      </div>

      <FilterBar
        role={resolvedRole}
        onRoleChange={setRole}
        startDate={startDate}
        endDate={endDate}
        onStartDateChange={setStartDate}
        onEndDateChange={setEndDate}
        statusCode={statusCode}
        onStatusCodeChange={setStatusCode}
      />

      <UsageTable
        role={resolvedRole}
        startDate={startDate || undefined}
        endDate={endDate || undefined}
        statusCode={statusCode !== " " ? statusCode : undefined}
      />
    </div>
  );
}
