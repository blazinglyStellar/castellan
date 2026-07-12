"use client"

import { useInfiniteQuery } from "@tanstack/react-query"
import type { CursorParams, PaginatedResponse } from "@/lib/api/types"

interface UseCursorPaginationOptions<T> {
  queryKey: string[]
  fetchFn: (params: CursorParams) => Promise<PaginatedResponse<T>>
  limit?: number
  refetchInterval?: number
}

interface UseCursorPaginationReturn<T> {
  items: T[]
  isLoading: boolean
  isLoadingMore: boolean
  hasMore: boolean
  loadMore: () => void
  refresh: () => void
  error: Error | null
}

export function useCursorPagination<T>({
  queryKey,
  fetchFn,
  limit = 50,
  refetchInterval,
}: UseCursorPaginationOptions<T>): UseCursorPaginationReturn<T> {
  const query = useInfiniteQuery<PaginatedResponse<T>>({
    queryKey: [...queryKey, { limit }],
    queryFn: ({ pageParam }) =>
      fetchFn({ cursor: pageParam as string | undefined, limit }),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => lastPage.next_cursor ?? undefined,
    refetchInterval,
  })

  return {
    items: query.data?.pages.flatMap((p) => p.data) ?? [],
    isLoading: query.isLoading,
    isLoadingMore: query.isFetchingNextPage,
    hasMore: !!query.hasNextPage,
    loadMore: query.fetchNextPage,
    refresh: query.refetch,
    error: query.error,
  }
}
