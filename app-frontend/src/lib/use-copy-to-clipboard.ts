"use client";

import { useState, useCallback } from "react";

export function useCopyToClipboard(resetMs = 2000) {
  const [copied, setCopied] = useState<string | null>(null);

  const copy = useCallback(
    async (text: string, label?: string) => {
      try {
        await navigator.clipboard.writeText(text);
        setCopied(label ?? text);
        setTimeout(() => setCopied(null), resetMs);
      } catch {
        setCopied(null);
      }
    },
    [resetMs],
  );

  return { copied, copy };
}
