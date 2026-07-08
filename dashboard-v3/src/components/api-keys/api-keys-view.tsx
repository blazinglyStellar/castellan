"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Plus,
  Copy,
  Check,
  EllipsisVertical,
  RotateCw,
  Ban,
} from "lucide-react";

import { EmptyState } from "@/components/ui/empty-state";
import { ErrorState } from "@/components/ui/error-state";

import {
  getApiKeys,
  createApiKey,
  revokeApiKey,
  rotateApiKey,
} from "@/lib/api/client";
import type { ApiKey } from "@/lib/api/types";
import { timeAgo, StatusBadge } from "@/lib/format";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

function maskKey(key: string): string {
  if (key.length <= 8) return key;
  const prefix = key.slice(0, 2);
  const rest = key.slice(2);
  return `${prefix}${rest.slice(0, 8)}`;
}

export function ApiKeysView() {
  const {
    data: keys,
    isLoading,
    isError,
    error,
    refetch,
  } = useQuery({
    queryKey: ["api-keys"],
    queryFn: getApiKeys,
  });

  if (isLoading) {
    return <LoadingSkeleton />;
  }

  if (isError) {
    return (
      <ErrorState
        message={
          error instanceof Error ? error.message : "Failed to load API keys"
        }
        onRetry={() => refetch()}
      />
    );
  }

  const hasKeys = keys && keys.length > 0;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <p className="text-sm text-muted-foreground">
            Manage API keys for authenticating requests.
          </p>
        </div>
        <GenerateKeyDialog />
      </div>

      {hasKeys ? (
        <KeysTable keys={keys} />
      ) : (
        <EmptyState
          title="No API keys yet"
          description="Generate your first API key to authenticate requests."
        />
      )}
    </div>
  );
}

function KeysTable({ keys }: { keys: ApiKey[] }) {
  const queryClient = useQueryClient();
  const [showKeyModal, setShowKeyModal] = useState<{
    key: string;
    title: string;
  } | null>(null);

  const revokeMutation = useMutation({
    mutationFn: revokeApiKey,
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: ["api-keys"] }),
  });

  const rotateMutation = useMutation({
    mutationFn: rotateApiKey,
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ["api-keys"] });
      setShowKeyModal({ key: data.key, title: "New Key Generated" });
    },
  });

  return (
    <>
      <Card>
        <CardHeader className="flex flex-row items-center gap-2">
          <CardTitle className="text-sm font-medium">API Keys</CardTitle>
        </CardHeader>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Label</TableHead>
                <TableHead>Key</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Created</TableHead>
                <TableHead>Expires</TableHead>
                <TableHead className="text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {keys.map((key) => (
                <TableRow key={key.id}>
                  <TableCell className="font-medium">
                    {key.label || "Unnamed"}
                  </TableCell>
                  <TableCell className="font-mono text-xs text-muted-foreground">
                    {maskKey(key.id)}
                  </TableCell>
                  <TableCell>
                    <StatusBadge status={key.status} />
                  </TableCell>
                  <TableCell className="whitespace-nowrap text-xs text-muted-foreground">
                    {timeAgo(key.created_at)}
                  </TableCell>
                  <TableCell className="whitespace-nowrap text-xs text-muted-foreground">
                    {key.expires_at
                      ? new Date(key.expires_at).toLocaleDateString()
                      : "—"}
                  </TableCell>
                  <TableCell className="text-right">
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button variant="ghost" size="icon">
                          <EllipsisVertical className="h-4 w-4 text-muted-foreground" />
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end">
                        {key.status === "active" && (
                          <DropdownMenuItem
                            onClick={() => rotateMutation.mutate(key.id)}
                            disabled={rotateMutation.isPending}
                          >
                            <RotateCw className="mr-2 h-4 w-4" />
                            Rotate
                          </DropdownMenuItem>
                        )}
                        {key.status === "active" && (
                          <DropdownMenuItem
                            onClick={() => revokeMutation.mutate(key.id)}
                            disabled={revokeMutation.isPending}
                          >
                            <Ban className="mr-2 h-4 w-4" />
                            Revoke
                          </DropdownMenuItem>
                        )}
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      <ShowKeyModal
        open={showKeyModal !== null}
        onOpenChange={(open) => {
          if (!open) setShowKeyModal(null);
        }}
        keyValue={showKeyModal?.key ?? ""}
        title={showKeyModal?.title ?? ""}
      />
    </>
  );
}

function GenerateKeyDialog() {
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  const [label, setLabel] = useState("");
  const [showKeyModal, setShowKeyModal] = useState<{
    key: string;
    title: string;
  } | null>(null);

  const mutation = useMutation({
    mutationFn: () => createApiKey(label.trim() || undefined),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ["api-keys"] });
      setOpen(false);
      setLabel("");
      setShowKeyModal({ key: data.key, title: "Key Generated" });
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    mutation.mutate();
  };

  return (
    <>
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogTrigger asChild>
          <Button size="sm">
            <Plus className="mr-2 h-4 w-4" />
            Generate Key
          </Button>
        </DialogTrigger>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Generate API Key</DialogTitle>
          </DialogHeader>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="label">Label (optional)</Label>
              <Input
                id="label"
                value={label}
                onChange={(e) => setLabel(e.target.value)}
                placeholder="My App Key"
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
                {mutation.isPending ? "Generating..." : "Generate"}
              </Button>
            </div>
          </form>
        </DialogContent>
      </Dialog>

      <ShowKeyModal
        open={showKeyModal !== null}
        onOpenChange={(open) => {
          if (!open) setShowKeyModal(null);
        }}
        keyValue={showKeyModal?.key ?? ""}
        title={showKeyModal?.title ?? ""}
      />
    </>
  );
}

function ShowKeyModal({
  open,
  onOpenChange,
  keyValue,
  title,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  keyValue: string;
  title: string;
}) {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    await navigator.clipboard.writeText(keyValue);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>
            Copy this key now. You will not be able to see it again.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          <div className="rounded-md bg-muted p-4">
            <code className="break-all font-mono text-sm">{keyValue}</code>
          </div>
          <div className="flex justify-end gap-2">
            <Button onClick={handleCopy} size="sm">
              {copied ? (
                <>
                  <Check className="mr-2 h-4 w-4" />
                  Copied
                </>
              ) : (
                <>
                  <Copy className="mr-2 h-4 w-4" />
                  Copy
                </>
              )}
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}

// ── States ──

function LoadingSkeleton() {
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-end">
        <Skeleton className="h-9 w-32 rounded-md" />
      </div>
      <Card>
        <CardHeader className="flex flex-row items-center gap-2">
          <Skeleton className="h-4 w-20" />
        </CardHeader>
        <CardContent className="p-0">
          <div className="space-y-0">
            {Array.from({ length: 3 }).map((_, i) => (
              <div
                key={i}
                className="flex items-center gap-4 border-t px-4 py-3"
              >
                <Skeleton className="h-3 w-20" />
                <Skeleton className="h-3 w-28" />
                <Skeleton className="h-4 w-16 rounded" />
                <Skeleton className="h-3 w-20" />
                <Skeleton className="h-3 w-20" />
                <Skeleton className="h-8 w-8" />
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}



// ── Helpers ──


