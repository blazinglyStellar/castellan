"use client"

import { Toaster as SonnerToaster, type ToasterProps } from "sonner"
import { useTheme } from "next-themes"

export function Toaster(props: ToasterProps) {
  const { resolvedTheme } = useTheme()
  return (
    <SonnerToaster
      theme={resolvedTheme as "light" | "dark"}
      {...props}
    />
  )
}
