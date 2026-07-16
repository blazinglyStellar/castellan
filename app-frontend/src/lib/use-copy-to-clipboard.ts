"use client";

import { useState, useCallback } from "react";
import { copyToClipboard } from "@/lib/clipboard";

export function useCopyToClipboard(resetMs = 2000) {
  const [copied, setCopied] = useState<string | null>(null);

  const copy = useCallback(
    async (text: string, label?: string) => {
      const ok = await copyToClipboard(text);
      setCopied(ok ? (label ?? text) : null);
      if (ok) setTimeout(() => setCopied(null), resetMs);
    },
    [resetMs],
  );

  return { copied, copy };
}
