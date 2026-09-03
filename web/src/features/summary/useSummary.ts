import { useQuery } from "@tanstack/react-query";
import { ApiError } from "../../lib/api-client";
import type { SummarySnapshot } from "../../types/api";
import { useAuth } from "../auth/AuthContext";

export function useSummary() {
  const { client, token } = useAuth();

  return useQuery<SummarySnapshot, ApiError>({
    queryKey: ["summary"],
    queryFn: ({ signal }) => client.summary(signal),
    enabled: Boolean(token),
    staleTime: 30_000,
    retry: false,
    refetchOnWindowFocus: false,
  });
}
