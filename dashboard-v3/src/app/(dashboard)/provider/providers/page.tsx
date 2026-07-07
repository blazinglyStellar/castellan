"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Plus, Inbox, RefreshCw, Trash2, Power, PowerOff } from "lucide-react";

import { useAccount } from "@/lib/auth/account-context";
import {
  getProviders,
  createProvider,
  deleteProvider,
  updateProviderStatus,
} from "@/lib/api/client";
import type { Provider, CreateProviderRequest } from "@/lib/api/types";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

export default function ProvidersPage() {
  const { isLoading: isAccountLoading } = useAccount();

  const {
    data: providers,
    isLoading,
    isError,
    error,
    refetch,
  } = useQuery({
    queryKey: ["providers"],
    queryFn: getProviders,
  });

  if (isAccountLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-muted border-t-primary" />
      </div>
    );
  }

  if (isLoading) {
    return <LoadingSkeleton />;
  }

  if (isError) {
    return (
      <ErrorState
        message={
          error instanceof Error ? error.message : "Failed to load providers"
        }
        onRetry={() => refetch()}
      />
    );
  }

  const hasProviders = providers && providers.length > 0;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Providers</h1>
          <p className="text-sm text-muted-foreground">
            Manage your registered API providers.
          </p>
        </div>
        <CreateProviderDialog />
      </div>

      {hasProviders ? (
        <ProvidersTable providers={providers} />
      ) : (
        <EmptyState />
      )}
    </div>
  );
}

function ProvidersTable({ providers }: { providers: Provider[] }) {
  const queryClient = useQueryClient();

  const deleteMutation = useMutation({
    mutationFn: deleteProvider,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["providers"] }),
  });

  const statusMutation = useMutation({
    mutationFn: ({ id, status }: { id: string; status: string }) =>
      updateProviderStatus(id, status),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["providers"] }),
  });

  return (
    <Card>
      <CardHeader className="flex flex-row items-center gap-2">
        <CardTitle className="text-sm font-medium">
          Registered Providers
        </CardTitle>
      </CardHeader>
      <CardContent className="p-0">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Name</TableHead>
              <TableHead>Base URL</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Created</TableHead>
              <TableHead className="text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {providers.map((provider) => (
              <TableRow key={provider.id}>
                <TableCell className="font-medium">
                  {provider.name}
                </TableCell>
                <TableCell className="max-w-[240px] truncate font-mono text-xs text-muted-foreground">
                  {provider.base_url}
                </TableCell>
                <TableCell>
                  <StatusBadge status={provider.status} />
                </TableCell>
                <TableCell className="whitespace-nowrap text-xs text-muted-foreground">
                  {timeAgo(provider.created_at)}
                </TableCell>
                <TableCell className="text-right">
                  <div className="flex items-center justify-end gap-1">
                    <Button
                      variant="ghost"
                      size="icon"
                      title={
                        provider.status === "active"
                          ? "Deactivate"
                          : "Activate"
                      }
                      onClick={() =>
                        statusMutation.mutate({
                          id: provider.id,
                          status:
                            provider.status === "active"
                              ? "inactive"
                              : "active",
                        })
                      }
                      disabled={statusMutation.isPending}
                    >
                      {provider.status === "active" ? (
                        <PowerOff className="h-4 w-4 text-muted-foreground" />
                      ) : (
                        <Power className="h-4 w-4 text-muted-foreground" />
                      )}
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      title="Delete"
                      onClick={() => deleteMutation.mutate(provider.id)}
                      disabled={deleteMutation.isPending}
                    >
                      <Trash2 className="h-4 w-4 text-muted-foreground hover:text-destructive" />
                    </Button>
                  </div>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
}

function CreateProviderDialog() {
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  const [name, setName] = useState("");
  const [baseUrl, setBaseUrl] = useState("");

  const mutation = useMutation({
    mutationFn: (data: CreateProviderRequest) => createProvider(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["providers"] });
      setOpen(false);
      setName("");
      setBaseUrl("");
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim() || !baseUrl.trim()) return;
    mutation.mutate({ name: name.trim(), base_url: baseUrl.trim() });
  };

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button size="sm">
          <Plus className="mr-2 h-4 w-4" />
          Add Provider
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Add Provider</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="name">Name</Label>
            <Input
              id="name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="My API Service"
              required
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="baseUrl">Base URL</Label>
            <Input
              id="baseUrl"
              value={baseUrl}
              onChange={(e) => setBaseUrl(e.target.value)}
              placeholder="https://api.example.com"
              type="url"
              required
            />
          </div>
          <div className="flex justify-end gap-2">
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => setOpen(false)}
            >
              Cancel
            </Button>
            <Button type="submit" size="sm" disabled={mutation.isPending}>
              {mutation.isPending ? "Adding..." : "Add Provider"}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  );
}

// ── States ──

function LoadingSkeleton() {
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <Skeleton className="h-7 w-24" />
          <Skeleton className="mt-1 h-4 w-56" />
        </div>
        <Skeleton className="h-9 w-32 rounded-md" />
      </div>
      <Card>
        <CardHeader className="flex flex-row items-center gap-2">
          <Skeleton className="h-4 w-32" />
        </CardHeader>
        <CardContent className="p-0">
          <div className="space-y-0">
            {Array.from({ length: 3 }).map((_, i) => (
              <div
                key={i}
                className="flex items-center gap-4 border-t px-4 py-3"
              >
                <Skeleton className="h-3 w-28" />
                <Skeleton className="h-3 w-44" />
                <Skeleton className="h-4 w-16 rounded" />
                <Skeleton className="h-3 w-20" />
                <Skeleton className="h-8 w-16" />
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

function ErrorState({
  message,
  onRetry,
}: {
  message: string;
  onRetry: () => void;
}) {
  return (
    <Card>
      <CardContent className="flex flex-col items-center gap-4 py-12">
        <p className="text-sm text-muted-foreground">{message}</p>
        <Button variant="outline" size="sm" onClick={onRetry}>
          <RefreshCw className="mr-2 h-3 w-3" />
          Retry
        </Button>
      </CardContent>
    </Card>
  );
}

function EmptyState() {
  return (
    <Card>
      <CardContent className="flex flex-col items-center gap-4 py-16 text-center">
        <Inbox className="h-8 w-8 text-muted-foreground" />
        <div>
          <p className="text-sm font-medium text-foreground">
            No providers yet
          </p>
          <p className="mt-1 text-sm text-muted-foreground">
            Add your first provider to start publishing APIs.
          </p>
        </div>
      </CardContent>
    </Card>
  );
}

// ── Helpers ──

function StatusBadge({ status }: { status: string }) {
  let color: string;
  switch (status) {
    case "active":
      color =
        "text-green-600 bg-green-100 dark:text-green-400 dark:bg-green-950";
      break;
    case "inactive":
      color =
        "text-gray-600 bg-gray-100 dark:text-gray-400 dark:bg-gray-950";
      break;
    default:
      color =
        "text-gray-600 bg-gray-100 dark:text-gray-400 dark:bg-gray-950";
  }

  return (
    <span
      className={`inline-block rounded px-1.5 py-0.5 font-mono text-[11px] capitalize ${color}`}
    >
      {status}
    </span>
  );
}

function timeAgo(timestamp: string): string {
  const now = Date.now();
  const then = new Date(timestamp).getTime();
  const diffSec = Math.floor((now - then) / 1000);

  if (diffSec < 60) return "just now";
  if (diffSec < 3600) return `${Math.floor(diffSec / 60)}m ago`;
  if (diffSec < 86400) return `${Math.floor(diffSec / 3600)}h ago`;
  return `${Math.floor(diffSec / 86400)}d ago`;
}
