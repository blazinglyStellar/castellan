"use client";

import { useState, useMemo } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import type { ColumnDef, SortingState, ColumnFiltersState } from "@tanstack/react-table";
import {
  flexRender,
  getCoreRowModel,
  getSortedRowModel,
  getFilteredRowModel,
  useReactTable,
} from "@tanstack/react-table";
import {
  Plus,
  Copy,
  Check,
  EllipsisVertical,
  RotateCw,
  Ban,
  Pencil,
  Clock,
  AlertTriangle,
  ArrowUpDown,
  ArrowUp,
  ArrowDown,
  Search,
  X,
} from "lucide-react";

import { ErrorState } from "@/components/shared/error-state";
import { EmptyState } from "@/components/shared/empty-state";

import {
  getApiKeys,
  createApiKey,
  updateApiKey,
  revokeApiKey,
  rotateApiKey,
} from "@/lib/api/endpoints";
import type { ApiKey, CreateApiKeyResponse } from "@/lib/api/types";
import { timeAgo, StatusBadge } from "@/lib/format";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
  DialogFooter,
} from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
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
import { cn } from "@/lib/utils";

// ── Helpers ──

function maskKey(keyId: string): string {
  const suffix = keyId.replace(/-/g, "").slice(0, 4);
  return `ca_****${suffix}`;
}

function KeyCell({ keyId }: { keyId: string }) {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    await navigator.clipboard.writeText(keyId);
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  };

  return (
    <span className="group flex items-center gap-1.5">
      <span className="font-mono text-xs text-muted-foreground">
        {maskKey(keyId)}
      </span>
      <button
        onClick={handleCopy}
        className="text-muted-foreground hover:text-foreground"
        title="Copy key ID"
      >
        {copied ? (
          <Check className="h-[18px] w-[18px] text-green-500" />
        ) : (
          <Copy className="h-[18px] w-[18px] text-muted-foreground hover:text-foreground" />
        )}
      </button>
    </span>
  );
}

const EXPIRATION_PRESETS = [
  { label: "30 days", days: 30 },
  { label: "90 days", days: 90 },
  { label: "1 year", days: 365 },
] as const;

function getExpiresAt(preset: string, customDate: string): string | null {
  if (preset === "custom" && customDate) {
    return new Date(customDate + "T23:59:59Z").toISOString();
  }
  const found = EXPIRATION_PRESETS.find((p) => p.label === preset);
  if (found) {
    const d = new Date();
    d.setDate(d.getDate() + found.days);
    return d.toISOString();
  }
  return null;
}

function formatExpires(expiresAt?: string): {
  text: string;
  urgent: boolean;
  expired: boolean;
  sortValue: number;
} {
  if (!expiresAt)
    return { text: "Never", urgent: false, expired: false, sortValue: Infinity };
  const d = new Date(expiresAt);
  const now = Date.now();
  const diffMs = d.getTime() - now;
  if (diffMs < 0)
    return { text: "Expired", urgent: false, expired: true, sortValue: -1 };
  const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));
  if (diffDays <= 7)
    return { text: `${diffDays}d remaining`, urgent: true, expired: false, sortValue: diffDays };
  if (diffDays <= 30)
    return { text: `${diffDays}d remaining`, urgent: true, expired: false, sortValue: diffDays };
  return {
    text: d.toLocaleDateString("en-US", { month: "short", day: "numeric", year: "numeric" }),
    urgent: false,
    expired: false,
    sortValue: diffDays,
  };
}

// ── Shared: Expiration Select ──

function ExpirationSelect({
  value,
  customDate,
  onChange,
  onCustomDateChange,
}: {
  value: string;
  customDate: string;
  onChange: (v: string) => void;
  onCustomDateChange: (v: string) => void;
}) {
  return (
    <div className="space-y-2">
      <Label>Expires</Label>
      <select
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
      >
        <option value="30 days">30 days</option>
        <option value="90 days">90 days</option>
        <option value="1 year">1 year</option>
        <option value="custom">Custom date</option>
      </select>
      {value === "custom" && (
        <Input
          type="date"
          value={customDate}
          onChange={(e) => onCustomDateChange(e.target.value)}
          min={new Date().toISOString().split("T")[0]}
          className="mt-1"
        />
      )}
    </div>
  );
}

// ── Main View ──

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
        <>
          <KeysSummaryBar keys={keys} />
          <KeysTable keys={keys} />
        </>
      ) : (
        <EmptyState
          title="No API keys yet"
          description="Generate your first API key to authenticate requests."
        />
      )}
    </div>
  );
}

// ── Summary Bar ──

function KeysSummaryBar({ keys }: { keys: ApiKey[] }) {
  const now = Date.now();
  const active = keys.filter((k) => k.status === "active").length;
  const expiringSoon = keys.filter((k) => {
    if (!k.expires_at || k.status !== "active") return false;
    const diffMs = new Date(k.expires_at).getTime() - now;
    return diffMs > 0 && diffMs <= 30 * 24 * 60 * 60 * 1000;
  }).length;
  const inactive = keys.filter((k) => k.status !== "active").length;

  return (
    <div className="flex gap-4 text-sm">
      <span>
        <strong className="text-foreground">{active}</strong>{" "}
        <span className="text-muted-foreground">active</span>
      </span>
      {expiringSoon > 0 && (
        <span className="flex items-center gap-1 text-amber-600 dark:text-amber-400">
          <AlertTriangle className="h-3.5 w-3.5" />
          <strong>{expiringSoon}</strong> expiring within 30 days
        </span>
      )}
      {inactive > 0 && (
        <span>
          <strong className="text-muted-foreground">{inactive}</strong>{" "}
          <span className="text-muted-foreground">revoked / expired</span>
        </span>
      )}
    </div>
  );
}

// ── Columns ──

const COLUMNS: ColumnDef<ApiKey>[] = [
  {
    id: "label",
    accessorKey: "label",
    header: ({ column }) => (
      <button
        className="flex items-center gap-1 font-medium"
        onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}
      >
        Label
        {{
          asc: <ArrowUp className="h-3 w-3" />,
          desc: <ArrowDown className="h-3 w-3" />,
        }[column.getIsSorted() as string] ?? (
          <ArrowUpDown className="h-3 w-3 text-muted-foreground/50" />
        )}
      </button>
    ),
    cell: ({ row }) => (
      <span className="font-medium">
        {row.original.label || (
          <span className="text-muted-foreground">Unnamed</span>
        )}
      </span>
    ),
    sortingFn: "text",
    filterFn: "includesString",
  },
  {
    id: "key",
    header: "Key",
    cell: ({ row }) => <KeyCell keyId={row.original.id} />,
    enableSorting: false,
  },
  {
    id: "status",
    accessorKey: "status",
    header: ({ column }) => (
      <button
        className="flex items-center gap-1 font-medium"
        onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}
      >
        Status
        {{
          asc: <ArrowUp className="h-3 w-3" />,
          desc: <ArrowDown className="h-3 w-3" />,
        }[column.getIsSorted() as string] ?? (
          <ArrowUpDown className="h-3 w-3 text-muted-foreground/50" />
        )}
      </button>
    ),
    cell: ({ row }) => <StatusBadge status={row.original.status} />,
    sortingFn: "text",
    filterFn: "equalsString",
  },
  {
    id: "created",
    accessorKey: "created_at",
    header: ({ column }) => (
      <button
        className="flex items-center gap-1 font-medium"
        onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}
      >
        Created
        {{
          asc: <ArrowUp className="h-3 w-3" />,
          desc: <ArrowDown className="h-3 w-3" />,
        }[column.getIsSorted() as string] ?? (
          <ArrowUpDown className="h-3 w-3 text-muted-foreground/50" />
        )}
      </button>
    ),
    cell: ({ row }) => (
      <span className="whitespace-nowrap text-xs text-muted-foreground">
        {timeAgo(row.original.created_at)}
      </span>
    ),
    sortingFn: "datetime",
  },
  {
    id: "expires",
    accessorFn: (row) => {
      const exp = formatExpires(row.expires_at);
      return exp.sortValue;
    },
    header: ({ column }) => (
      <button
        className="flex items-center gap-1 font-medium"
        onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}
      >
        Expires
        {{
          asc: <ArrowUp className="h-3 w-3" />,
          desc: <ArrowDown className="h-3 w-3" />,
        }[column.getIsSorted() as string] ?? (
          <ArrowUpDown className="h-3 w-3 text-muted-foreground/50" />
        )}
      </button>
    ),
    cell: ({ row }) => {
      const exp = formatExpires(row.original.expires_at);
      return (
        <span
          className={cn(
            "whitespace-nowrap text-xs",
            exp.expired && "text-red-500",
            exp.urgent && !exp.expired && "text-amber-600 dark:text-amber-400",
            !exp.expired && !exp.urgent && "text-muted-foreground",
          )}
        >
          {exp.text}
          {exp.urgent && !exp.expired && (
            <AlertTriangle className="ml-1 inline h-3 w-3" />
          )}
        </span>
      );
    },
    enableSorting: true,
  },
  {
    id: "actions",
    header: "",
    cell: ({ row }) => <RowActions keyData={row.original} />,
    enableSorting: false,
    enableColumnFilter: false,
  },
];

// ── Row Actions ──

function RowActions({ keyData }: { keyData: ApiKey }) {
  const queryClient = useQueryClient();
  const [showKeyModal, setShowKeyModal] = useState<{
    key: string;
    title: string;
    expiresAt?: string;
  } | null>(null);
  const [editingLabel, setEditingLabel] = useState(false);
  const [editingExpiration, setEditingExpiration] = useState(false);
  const [revoking, setRevoking] = useState(false);

  const revokeMutation = useMutation({
    mutationFn: () => revokeApiKey(keyData.id),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: ["api-keys"] }),
  });

  const rotateMutation = useMutation({
    mutationFn: () => rotateApiKey(keyData.id),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ["api-keys"] });
      setShowKeyModal({
        key: data.key,
        title: "New Key Generated",
        expiresAt: data.expires_at,
      });
    },
  });

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger>
          <Button variant="ghost" size="icon">
            <EllipsisVertical className="h-4 w-4 text-muted-foreground" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          {keyData.status === "active" && (
            <>
              <DropdownMenuItem onClick={() => setEditingLabel(true)}>
                <Pencil className="mr-2 h-4 w-4" />
                Edit Label
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => setEditingExpiration(true)}>
                <Clock className="mr-2 h-4 w-4" />
                Set Expiration
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem
                onClick={() => rotateMutation.mutate()}
                disabled={rotateMutation.isPending}
              >
                <RotateCw className="mr-2 h-4 w-4" />
                Rotate
              </DropdownMenuItem>
              <DropdownMenuItem
                onClick={() => setRevoking(true)}
                disabled={revokeMutation.isPending}
                className="text-red-500 focus:text-red-500"
              >
                <Ban className="mr-2 h-4 w-4" />
                Revoke
              </DropdownMenuItem>
            </>
          )}
          {keyData.status !== "active" && (
            <DropdownMenuItem onClick={() => setEditingLabel(true)}>
              <Pencil className="mr-2 h-4 w-4" />
              Edit Label
            </DropdownMenuItem>
          )}
        </DropdownMenuContent>
      </DropdownMenu>

      <ShowKeyModal
        open={showKeyModal !== null}
        onOpenChange={(open) => {
          if (!open) setShowKeyModal(null);
        }}
        keyValue={showKeyModal?.key ?? ""}
        title={showKeyModal?.title ?? ""}
        expiresAt={showKeyModal?.expiresAt}
      />

      <EditLabelDialog
        keyData={keyData}
        open={editingLabel}
        onClose={() => setEditingLabel(false)}
      />
      <SetExpirationDialog
        keyData={keyData}
        open={editingExpiration}
        onClose={() => setEditingExpiration(false)}
      />
      <RevokeConfirmDialog
        keyData={revoking ? keyData : null}
        onClose={() => setRevoking(false)}
        onConfirm={() => {
          revokeMutation.mutate();
          setRevoking(false);
        }}
        isPending={revokeMutation.isPending}
      />
    </>
  );
}

// ── Table ──

function KeysTable({ keys }: { keys: ApiKey[] }) {
  const [sorting, setSorting] = useState<SortingState>([]);
  const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>([]);

  const table = useReactTable({
    data: keys,
    columns: COLUMNS,
    state: { sorting, columnFilters },
    onSortingChange: setSorting,
    onColumnFiltersChange: setColumnFilters,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
  });

  const labelFilterValue =
    (table.getColumn("label")?.getFilterValue() as string) ?? "";
  const statusFilterValue =
    (table.getColumn("status")?.getFilterValue() as string) ?? "";

  return (
    <Card>
      <CardHeader className="flex flex-row items-center gap-2 pb-3">
        <CardTitle className="text-sm font-medium">API Keys</CardTitle>
        <span className="text-xs text-muted-foreground">
          {table.getFilteredRowModel().rows.length} of {keys.length}
        </span>
      </CardHeader>
      <CardContent className="p-0">
        <div className="flex items-center gap-3 border-b px-4 py-3">
          <div className="relative flex-1">
            <Search className="absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              placeholder="Filter by label..."
              value={labelFilterValue}
              onChange={(e) =>
                table.getColumn("label")?.setFilterValue(e.target.value)
              }
              className="h-8 pl-8 text-xs"
            />
            {labelFilterValue && (
              <button
                onClick={() => table.getColumn("label")?.setFilterValue("")}
                className="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
              >
                <X className="h-3.5 w-3.5" />
              </button>
            )}
          </div>
          <Select
            value={statusFilterValue}
            onValueChange={(v) =>
              table
                .getColumn("status")
                ?.setFilterValue(v === "all" ? "" : v)
            }
          >
            <SelectTrigger className="h-8 w-[140px] text-xs">
              <SelectValue placeholder="All statuses" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all" className="text-xs">
                All statuses
              </SelectItem>
              <SelectItem value="active" className="text-xs">
                Active
              </SelectItem>
              <SelectItem value="revoked" className="text-xs">
                Revoked
              </SelectItem>
              <SelectItem value="expired" className="text-xs">
                Expired
              </SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div className="relative w-full overflow-auto">
          <Table>
            <TableHeader>
              {table.getHeaderGroups().map((headerGroup) => (
                <TableRow key={headerGroup.id}>
                  {headerGroup.headers.map((header) => (
                    <TableHead key={header.id}>
                      {flexRender(
                        header.column.columnDef.header,
                        header.getContext(),
                      )}
                    </TableHead>
                  ))}
                </TableRow>
              ))}
            </TableHeader>
            <TableBody>
              {table.getRowModel().rows.length ? (
                table.getRowModel().rows.map((row) => (
                  <TableRow key={row.id}>
                    {row.getVisibleCells().map((cell) => (
                      <TableCell key={cell.id}>
                        {flexRender(
                          cell.column.columnDef.cell,
                          cell.getContext(),
                        )}
                      </TableCell>
                    ))}
                  </TableRow>
                ))
              ) : (
                <TableRow>
                  <TableCell
                    colSpan={COLUMNS.length}
                    className="h-24 text-center"
                  >
                    <div className="flex flex-col items-center gap-1">
                      <span className="text-sm text-muted-foreground">
                        No keys match your filters
                      </span>
                      <Button
                        variant="link"
                        size="sm"
                        onClick={() => setColumnFilters([])}
                        className="h-auto p-0 text-xs"
                      >
                        Clear filters
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </div>
      </CardContent>
    </Card>
  );
}

// ── Generate Key Dialog ──

function GenerateKeyDialog() {
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  const [label, setLabel] = useState("");
  const [expPreset, setExpPreset] = useState("90 days");
  const [customDate, setCustomDate] = useState("");
  const [showKeyModal, setShowKeyModal] = useState<{
    key: string;
    title: string;
    expiresAt?: string;
  } | null>(null);

  const mutation = useMutation({
    mutationFn: () => {
      const expiresAt = getExpiresAt(expPreset, customDate);
      return createApiKey(label.trim() || "Unnamed", expiresAt ?? undefined);
    },
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ["api-keys"] });
      setOpen(false);
      setLabel("");
      setExpPreset("90 days");
      setCustomDate("");
      setShowKeyModal({
        key: data.key,
        title: "Key Generated",
        expiresAt: data.expires_at,
      });
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    mutation.mutate();
  };

  return (
    <>
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogTrigger>
          <Button size="sm">
            <Plus className="mr-2 h-4 w-4" />
            Generate Key
          </Button>
        </DialogTrigger>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Generate API Key</DialogTitle>
            <DialogDescription>
              Create a new API key for authenticating requests through the
              gateway.
            </DialogDescription>
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
            <ExpirationSelect
              value={expPreset}
              customDate={customDate}
              onChange={setExpPreset}
              onCustomDateChange={setCustomDate}
            />
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
        expiresAt={showKeyModal?.expiresAt}
      />
    </>
  );
}

// ── Edit Label Dialog ──

function EditLabelDialog({
  keyData,
  open,
  onClose,
}: {
  keyData: ApiKey;
  open: boolean;
  onClose: () => void;
}) {
  const queryClient = useQueryClient();
  const [label, setLabel] = useState("");

  const mutation = useMutation({
    mutationFn: () =>
      updateApiKey(keyData.id, { label: label.trim() || "Unnamed" }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["api-keys"] });
      onClose();
    },
  });

  return (
    <Dialog open={open} onOpenChange={(open) => !open && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Edit Label</DialogTitle>
          <DialogDescription>
            Rename this API key for easier identification.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="edit-label">Label</Label>
            <Input
              id="edit-label"
              defaultValue={keyData.label || ""}
              onChange={(e) => setLabel(e.target.value)}
              placeholder="My App Key"
            />
          </div>
          <div className="flex justify-end gap-2">
            <Button variant="outline" size="sm" onClick={onClose}>
              Cancel
            </Button>
            <Button
              size="sm"
              onClick={() => mutation.mutate()}
              disabled={mutation.isPending}
            >
              {mutation.isPending ? "Saving..." : "Save"}
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}

// ── Set Expiration Dialog ──

function SetExpirationDialog({
  keyData,
  open,
  onClose,
}: {
  keyData: ApiKey;
  open: boolean;
  onClose: () => void;
}) {
  const queryClient = useQueryClient();
  const [expPreset, setExpPreset] = useState("90 days");
  const [customDate, setCustomDate] = useState("");

  const mutation = useMutation({
    mutationFn: () => {
      const expiresAt = getExpiresAt(expPreset, customDate);
      return updateApiKey(keyData.id, { expires_at: expiresAt });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["api-keys"] });
      onClose();
    },
  });

  const currentExp = keyData.expires_at
    ? new Date(keyData.expires_at).toLocaleDateString()
    : "never";

  return (
    <Dialog open={open} onOpenChange={(open) => !open && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Set Expiration</DialogTitle>
          <DialogDescription>
            Change when this key expires. Currently:{" "}
            <strong>{currentExp}</strong>.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          <ExpirationSelect
            value={expPreset}
            customDate={customDate}
            onChange={setExpPreset}
            onCustomDateChange={setCustomDate}
          />
          <div className="flex justify-end gap-2">
            <Button variant="outline" size="sm" onClick={onClose}>
              Cancel
            </Button>
            <Button
              size="sm"
              onClick={() => mutation.mutate()}
              disabled={mutation.isPending}
            >
              {mutation.isPending ? "Saving..." : "Save"}
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}

// ── Revoke Confirm Dialog ──

function RevokeConfirmDialog({
  keyData,
  onClose,
  onConfirm,
  isPending,
}: {
  keyData: ApiKey | null;
  onClose: () => void;
  onConfirm: () => void;
  isPending: boolean;
}) {
  if (!keyData) return null;

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Revoke API Key</DialogTitle>
          <DialogDescription>
            Are you sure you want to revoke{" "}
            <strong>{keyData.label || "Unnamed"}</strong>? This action cannot be
            undone. Any services using this key will lose access immediately.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter className="gap-2">
          <Button variant="outline" size="sm" onClick={onClose}>
            Cancel
          </Button>
          <Button
            variant="destructive"
            size="sm"
            onClick={onConfirm}
            disabled={isPending}
          >
            {isPending ? "Revoking..." : "Revoke"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// ── Show Key Modal ──

function ShowKeyModal({
  open,
  onOpenChange,
  keyValue,
  title,
  expiresAt,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  keyValue: string;
  title: string;
  expiresAt?: string;
}) {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    await navigator.clipboard.writeText(keyValue);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const expText = expiresAt
    ? new Date(expiresAt).toLocaleDateString("en-US", {
        month: "long",
        day: "numeric",
        year: "numeric",
      })
    : null;

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
          {expText && (
            <p className="text-xs text-muted-foreground">
              Expires:{" "}
              <span className="font-medium text-foreground">{expText}</span>
            </p>
          )}
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
