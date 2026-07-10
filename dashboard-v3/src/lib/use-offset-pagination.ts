"use client";

import { useQuery } from "@tanstack/react-query";
import { useCallback, useMemo, useState } from "react";

interface UseOffsetPaginationOptions<T> {
  queryKey: string[];
  fetchFn: (params: {
    offset: number;
    limit: number;
  }) => Promise<{ data: T[]; total: number }>;
  initialPage?: number;
  initialPageSize?: number;
}

interface UseOffsetPaginationReturn<T> {
  items: T[];
  isLoading: boolean;
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
  setPage: (page: number) => void;
  setPageSize: (size: number) => void;
  refresh: () => void;
  error: Error | null;
}

export function useOffsetPagination<T>({
  queryKey,
  fetchFn,
  initialPage = 1,
  initialPageSize = 20,
}: UseOffsetPaginationOptions<T>): UseOffsetPaginationReturn<T> {
  const [page, setPage] = useState(initialPage);
  const [pageSize, setPageSize] = useState(initialPageSize);

  const offset = (page - 1) * pageSize;

  const query = useQuery({
    queryKey: [...queryKey, { page, pageSize, offset }],
    queryFn: () => fetchFn({ offset, limit: pageSize }),
  });

  const totalPages = useMemo(
    () => Math.max(1, Math.ceil((query.data?.total ?? 0) / pageSize)),
    [query.data?.total, pageSize],
  );

  const goToPage = useCallback(
    (nextPage: number) => {
      setPage(Math.max(1, Math.min(nextPage, totalPages)));
    },
    [totalPages],
  );

  const changePageSize = useCallback(
    (newSize: number) => {
      setPageSize(newSize);
      setPage(1);
    },
    [],
  );

  return {
    items: query.data?.data ?? [],
    isLoading: query.isLoading,
    total: query.data?.total ?? 0,
    page,
    pageSize,
    totalPages,
    setPage: goToPage,
    setPageSize: changePageSize,
    refresh: query.refetch,
    error: query.error,
  };
}
