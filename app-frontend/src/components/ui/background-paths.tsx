"use client"

import { motion } from "motion/react"
import { memo, useId, useMemo, type ReactNode } from "react"

interface Point {
  x: number
  y: number
}

function generateAestheticPath(
  index: number,
  position: number,
  type: "primary" | "secondary" | "accent",
): string {
  const baseAmplitude =
    type === "primary" ? 150 : type === "secondary" ? 100 : 60
  const phase = index * 0.2
  const points: Point[] = []
  const segments = type === "primary" ? 10 : type === "secondary" ? 8 : 6

  const startX = 2400
  const startY = 800
  const endX = -2400
  const endY = -800 + index * 25

  for (let i = 0; i <= segments; i++) {
    const progress = i / segments
    const eased = 1 - (1 - progress) ** 2

    const baseX = startX + (endX - startX) * eased
    const baseY = startY + (endY - startY) * eased

    const amplitudeFactor = 1 - eased * 0.3
    const wave1 =
      Math.sin(progress * Math.PI * 3 + phase) *
      (baseAmplitude * 0.7 * amplitudeFactor)
    const wave2 =
      Math.cos(progress * Math.PI * 4 + phase) *
      (baseAmplitude * 0.3 * amplitudeFactor)
    const wave3 =
      Math.sin(progress * Math.PI * 2 + phase) *
      (baseAmplitude * 0.2 * amplitudeFactor)

    points.push({
      x: baseX * position,
      y: baseY + wave1 + wave2 + wave3,
    })
  }

  const pathCommands = points.map((point: Point, i: number) => {
    if (i === 0) return `M ${point.x.toFixed(4)} ${point.y.toFixed(4)}`
    const prevPoint = points[i - 1]
    const tension = 0.4
    const cp1x = prevPoint.x + (point.x - prevPoint.x) * tension
    const cp1y = prevPoint.y
    const cp2x = prevPoint.x + (point.x - prevPoint.x) * (1 - tension)
    const cp2y = point.y
    return `C ${cp1x.toFixed(4)} ${cp1y.toFixed(4)}, ${cp2x.toFixed(4)} ${cp2y.toFixed(4)}, ${point.x.toFixed(4)} ${point.y.toFixed(4)}`
  })

  return pathCommands.join(" ")
}

const FloatingPaths = memo(function FloatingPaths({
  position,
}: {
  position: number
}) {
  const baseId = useId()

  const primaryPaths = useMemo(
    () =>
      Array.from({ length: 12 }, (_, i) => ({
        d: generateAestheticPath(i, position, "primary"),
        opacity: 0.15 + i * 0.02,
        width: 4 + i * 0.3,
      })),
    [position],
  )

  const secondaryPaths = useMemo(
    () =>
      Array.from({ length: 15 }, (_, i) => ({
        d: generateAestheticPath(i, position, "secondary"),
        opacity: 0.12 + i * 0.015,
        width: 3 + i * 0.25,
      })),
    [position],
  )

  const accentPaths = useMemo(
    () =>
      Array.from({ length: 10 }, (_, i) => ({
        d: generateAestheticPath(i, position, "accent"),
        opacity: 0.08 + i * 0.12,
        width: 2 + i * 0.2,
      })),
    [position],
  )

  return (
    <div className="pointer-events-none absolute inset-0 overflow-hidden">
      <svg
        className="h-full w-full text-slate-950/40 dark:text-white/40"
        fill="none"
        preserveAspectRatio="xMidYMid slice"
        viewBox="-2400 -800 4800 1600"
      >
        <title>Background Paths</title>
        <defs>
          <linearGradient id="sharedGradient" x1="0%" x2="100%" y1="0%" y2="0%">
            <stop offset="0%" stopColor="rgba(147, 51, 234, 0.5)" />
            <stop offset="50%" stopColor="rgba(236, 72, 153, 0.5)" />
            <stop offset="100%" stopColor="rgba(59, 130, 246, 0.5)" />
          </linearGradient>
        </defs>

        <g className="primary-waves">
          {primaryPaths.map((path, i) => (
            <motion.path
              key={`${baseId}-primary-${i}`}
              d={path.d}
              stroke="url(#sharedGradient)"
              strokeLinecap="round"
              strokeWidth={path.width}
              initial={{ opacity: 0, scale: 0.8 }}
              animate={{ opacity: 1, scale: 1, y: [0, -15, 0] }}
              style={{ opacity: path.opacity }}
              transition={{
                opacity: { duration: 1 },
                scale: { duration: 1 },
                y: {
                  duration: 4,
                  repeat: Number.POSITIVE_INFINITY,
                  ease: "easeInOut",
                  repeatType: "reverse",
                },
              }}
            />
          ))}
        </g>

        <g className="secondary-waves" style={{ opacity: 0.8 }}>
          {secondaryPaths.map((path, i) => (
            <motion.path
              key={`${baseId}-secondary-${i}`}
              d={path.d}
              stroke="url(#sharedGradient)"
              strokeLinecap="round"
              strokeWidth={path.width}
              initial={{ opacity: 0, scale: 0.9 }}
              animate={{ opacity: 1, scale: 1, y: [0, -10, 0] }}
              style={{ opacity: path.opacity }}
              transition={{
                opacity: { duration: 1 },
                scale: { duration: 1 },
                y: {
                  duration: 3,
                  repeat: Number.POSITIVE_INFINITY,
                  ease: "easeInOut",
                  repeatType: "reverse",
                },
              }}
            />
          ))}
        </g>

        <g className="accent-waves" style={{ opacity: 0.6 }}>
          {accentPaths.map((path, i) => (
            <motion.path
              key={`${baseId}-accent-${i}`}
              d={path.d}
              stroke="url(#sharedGradient)"
              strokeLinecap="round"
              strokeWidth={path.width}
              initial={{ opacity: 0, scale: 0.95 }}
              animate={{ opacity: 1, scale: 1, y: [0, -5, 0] }}
              style={{ opacity: path.opacity }}
              transition={{
                opacity: { duration: 1 },
                scale: { duration: 1 },
                y: {
                  duration: 2,
                  repeat: Number.POSITIVE_INFINITY,
                  ease: "easeInOut",
                  repeatType: "reverse",
                },
              }}
            />
          ))}
        </g>
      </svg>
    </div>
  )
})

export { FloatingPaths }

export default memo(function BackgroundPaths({
  title = "Background Paths",
  children,
}: {
  title?: string
  children?: ReactNode
}) {
  if (children) {
    return (
      <div className="relative flex min-h-screen w-full items-center justify-center overflow-hidden bg-white dark:bg-neutral-950">
        <div className="pointer-events-none absolute inset-0 overflow-hidden">
          <FloatingPaths position={1} />
        </div>
        <div className="relative z-10">{children}</div>
      </div>
    )
  }

  return (
    <div className="relative flex min-h-screen w-full items-center justify-center overflow-hidden bg-white dark:bg-neutral-950">
      <div className="pointer-events-none absolute inset-0 overflow-hidden">
        <FloatingPaths position={1} />
      </div>

      <div className="container relative z-10 mx-auto px-4 text-center md:px-6">
        <motion.div
          animate={{ opacity: 1 }}
          className="mx-auto max-w-4xl"
          initial={{ opacity: 0 }}
          transition={{ duration: 2 }}
        >
          <motion.h1
            animate={{ opacity: 1, y: 0 }}
            className="mb-8 bg-gradient-to-r from-neutral-800/90 to-neutral-600/90 bg-clip-text font-bold text-3xl text-transparent tracking-tighter sm:text-5xl md:text-5xl dark:from-white/90 dark:to-white/70"
            initial={{ opacity: 0, y: 20 }}
            transition={{
              duration: 1.2,
              ease: [0.2, 0.65, 0.3, 0.9],
            }}
          >
            {title}
          </motion.h1>
        </motion.div>
      </div>
    </div>
  )
})
