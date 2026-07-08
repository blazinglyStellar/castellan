"use client";

import { useState } from "react";
import { useParams } from "next/navigation";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Plus,
  Trash2,
  Power,
  PowerOff,
  ArrowLeft,
} from "lucide-react";
import Link from "next/link";

import { EmptyState } from "@/components/ui/empty-state";
import { ErrorState } from "@/components/ui/error-state";

import { useAccount } from "@/lib/auth/account-context";
import {
  getProviderEndpoints,
  createEndpoint,
  deleteEndpoint,
  updateEndpointStatus,
} from "@/lib/api/client";
import type { Endpoint, CreateEndpointRequest } from "@/lib/api/types";
import { timeAgo, StatusBadge } from "@/lib/format";
import { MethodBadge } from "@/components/usage/method-badge";
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
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

export default function EndpointsPage() {
  const { isLoading: isAccountLoading } = useAccount();
  const params = useParams();
  const providerId = params.id as string;

  const {
    data: endpoints,
    isLoading,
    isError,
    error,
    refetch,
  } = useQuery({
    queryKey: ["endpoints", providerId],
    queryFn: () => getProviderEndpoints(providerId),
    enabled: !!providerId,
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
          error instanceof Error ? error.message : "Failed to load endpoints"
        }
        onRetry={() => refetch()}
      />
    );
  }

  const hasEndpoints = endpoints && endpoints.length > 0;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Link
            href="/provider/providers"
            className="rounded-md p-1 text-muted-foreground transition-colors hover:text-foreground"
          >
            <ArrowLeft className="h-5 w-5" />
          </Link>
          <div>
            <h1 className="text-2xl font-semibold tracking-tight">Endpoints</h1>
            <p className="text-sm text-muted-foreground">
              Configure endpoint routes, pricing, and rate limits.
            </p>
          </div>
        </div>
        <CreateEndpointDialog providerId={providerId} />
      </div>

      {hasEndpoints ? (
        <EndpointsTable endpoints={endpoints} />
      ) : (
        <EmptyState
          title="No endpoints yet"
          description="Add your first endpoint to define pricing and rate limits."
        />
      )}
    </div>
  );
}

function EndpointsTable({ endpoints }: { endpoints: Endpoint[] }) {
  const queryClient = useQueryClient();

  const deleteMutation = useMutation({
    mutationFn: deleteEndpoint,
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: ["endpoints"] }),
  });

  const statusMutation = useMutation({
    mutationFn: ({ id, status }: { id: string; status: string }) =>
      updateEndpointStatus(id, status),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: ["endpoints"] }),
  });

  return (
    <Card>
      <CardHeader className="flex flex-row items-center gap-2">
        <CardTitle className="text-sm font-medium">
          Configured Endpoints
        </CardTitle>
      </CardHeader>
      <CardContent className="p-0">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Method</TableHead>
              <TableHead>Route</TableHead>
              <TableHead>Price</TableHead>
              <TableHead>Currency</TableHead>
              <TableHead>Rate Limit</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Created</TableHead>
              <TableHead className="text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {endpoints.map((ep) => (
              <TableRow key={ep.id}>
                <TableCell>
                  <MethodBadge method={ep.method} />
                </TableCell>
                <TableCell className="max-w-[200px] truncate font-mono text-xs">
                  {ep.route}
                </TableCell>
                <TableCell className="font-mono text-xs">
                  {ep.price_amount}
                </TableCell>
                <TableCell className="font-mono text-xs">
                  {ep.currency}
                </TableCell>
                <TableCell className="font-mono text-xs">
                  {ep.rate_limit ?? "—"}
                </TableCell>
                <TableCell>
                  <StatusBadge status={ep.status} />
                </TableCell>
                <TableCell className="whitespace-nowrap text-xs text-muted-foreground">
                  {timeAgo(ep.created_at)}
                </TableCell>
                <TableCell className="text-right">
                  <div className="flex items-center justify-end gap-1">
                    <Button
                      variant="ghost"
                      size="icon"
                      title={
                        ep.status === "active" ? "Deactivate" : "Activate"
                      }
                      onClick={() =>
                        statusMutation.mutate({
                          id: ep.id,
                          status:
                            ep.status === "active" ? "inactive" : "active",
                        })
                      }
                      disabled={statusMutation.isPending}
                    >
                      {ep.status === "active" ? (
                        <PowerOff className="h-4 w-4 text-muted-foreground" />
                      ) : (
                        <Power className="h-4 w-4 text-muted-foreground" />
                      )}
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      title="Delete"
                      onClick={() => deleteMutation.mutate(ep.id)}
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

function CreateEndpointDialog({
  providerId,
}: {
  providerId: string;
}) {
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  const [route, setRoute] = useState("");
  const [method, setMethod] = useState("GET");
  const [priceAmount, setPriceAmount] = useState("");
  const [currency, setCurrency] = useState("XLM");
  const [rateLimit, setRateLimit] = useState("");

  const mutation = useMutation({
    mutationFn: (data: CreateEndpointRequest) =>
      createEndpoint(providerId, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["endpoints"] });
      setOpen(false);
      setRoute("");
      setMethod("GET");
      setPriceAmount("");
      setCurrency("XLM");
      setRateLimit("");
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!route.trim() || !priceAmount.trim() || !currency.trim()) return;
    mutation.mutate({
      route: route.trim(),
      method,
      price_amount: priceAmount.trim(),
      currency: currency.trim(),
      rate_limit: rateLimit ? Number(rateLimit) : undefined,
    });
  };

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button size="sm">
          <Plus className="mr-2 h-4 w-4" />
          Add Endpoint
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Add Endpoint</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="method">Method</Label>
            <Select value={method} onValueChange={setMethod}>
              <SelectTrigger id="method">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {["GET", "POST", "PUT", "PATCH", "DELETE"].map((m) => (
                  <SelectItem key={m} value={m}>
                    {m}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-2">
            <Label htmlFor="route">Route</Label>
            <Input
              id="route"
              value={route}
              onChange={(e) => setRoute(e.target.value)}
              placeholder="/v1/chat/completions"
              required
            />
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="priceAmount">Price Amount</Label>
              <Input
                id="priceAmount"
                value={priceAmount}
                onChange={(e) => setPriceAmount(e.target.value)}
                placeholder="0.01"
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="currency">Currency</Label>
              <Input
                id="currency"
                value={currency}
                onChange={(e) => setCurrency(e.target.value)}
                placeholder="XLM"
                required
              />
            </div>
          </div>
          <div className="space-y-2">
            <Label htmlFor="rateLimit">Rate Limit (requests/s, optional)</Label>
            <Input
              id="rateLimit"
              type="number"
              min="0"
              value={rateLimit}
              onChange={(e) => setRateLimit(e.target.value)}
              placeholder="100"
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
              {mutation.isPending ? "Adding..." : "Add Endpoint"}
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
        <div className="flex items-center gap-4">
          <Skeleton className="h-5 w-5 rounded" />
          <div>
            <Skeleton className="h-7 w-24" />
            <Skeleton className="mt-1 h-4 w-64" />
          </div>
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
                <Skeleton className="h-4 w-12 rounded" />
                <Skeleton className="h-3 w-36" />
                <Skeleton className="h-3 w-14" />
                <Skeleton className="h-3 w-10" />
                <Skeleton className="h-3 w-10" />
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



// ── Helpers ──


