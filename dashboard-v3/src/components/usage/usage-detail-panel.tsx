"use client";

import { useEffect, useRef } from "react";
import Image from "next/image";
import { X } from "lucide-react";

import type { UsageEvent } from "@/lib/api/types";
import { formatBytes, StatusCodeBadge, UsageStatusBadge } from "@/lib/format";
import { MethodBadge } from "@/components/usage/method-badge";
import { Button } from "@/components/ui/button";

interface UsageDetailPanelProps {
  event: UsageEvent | null;
  onClose: () => void;
}

export function UsageDetailPanel({ event, onClose }: UsageDetailPanelProps) {
  const panelRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handleKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    document.addEventListener("keydown", handleKey);
    return () => document.removeEventListener("keydown", handleKey);
  }, [onClose]);

  useEffect(() => {
    if (event && panelRef.current) {
      panelRef.current.focus();
    }
  }, [event]);

  if (!event) return null;

  const formattedTime = new Date(event.timestamp).toLocaleString("en-US", {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    timeZoneName: "short",
  });

  return (
    <>
      <div
        className="fixed inset-0 z-40 bg-black/20 backdrop-blur-sm"
        onClick={onClose}
      />
      <div
        ref={panelRef}
        tabIndex={-1}
        className="fixed inset-y-0 right-0 z-50 w-full max-w-md border-l bg-background shadow-xl outline-none"
      >
        <div className="flex h-full flex-col">
          <div className="flex items-center justify-between border-b px-6 py-4">
            <h2 className="text-base font-semibold">Request Details</h2>
            <Button variant="ghost" size="icon" onClick={onClose}>
              <X className="h-4 w-4" />
            </Button>
          </div>

          <div className="flex-1 space-y-6 overflow-y-auto px-6 py-6">
            <DetailRow label="Request ID" mono>
              {event.request_id}
            </DetailRow>

            <DetailRow label="Provider">
              {event.provider_name || event.provider_id}
            </DetailRow>

            <DetailRow label="Endpoint">
              <div className="flex items-center gap-2">
                <MethodBadge method={event.method} />
                <span className="font-mono text-xs">{event.route}</span>
              </div>
            </DetailRow>

            <DetailRow label="Cost">
              <span className="font-mono font-medium">
                {event.request_cost}{" "}
                <Image
                  src="/stellar-xlm-logo.svg"
                  alt="XLM"
                  width={14}
                  height={12}
                  className="inline-block align-middle"
                />
              </span>
            </DetailRow>

            <DetailRow label="Status Code">
              <StatusCodeBadge code={event.status_code} />
            </DetailRow>

            <DetailRow label="Usage Status">
              <UsageStatusBadge status={event.usage_status} />
            </DetailRow>

            <DetailRow label="Latency">
              <span className="font-mono">
                {event.latency_ms != null ? `${event.latency_ms}ms` : "\u2014"}
              </span>
            </DetailRow>

            <DetailRow label="Response Size">
              <span className="font-mono">
                {formatBytes(event.response_size)}
              </span>
            </DetailRow>

            <DetailRow label="Timestamp">
              <span className="text-sm">{formattedTime}</span>
            </DetailRow>
          </div>
        </div>
      </div>
    </>
  );
}

function DetailRow({
  label,
  mono,
  children,
}: {
  label: string;
  mono?: boolean;
  children: React.ReactNode;
}) {
  return (
    <div>
      <dt className="mb-1 text-xs font-medium text-muted-foreground uppercase tracking-wider">
        {label}
      </dt>
      <dd className={mono ? "font-mono text-xs break-all" : "text-sm"}>
        {children}
      </dd>
    </div>
  );
}
