// DesignDocument/5_FrontendDesign.md 5.7 APIクライアント設計
import { supabase } from "../supabase/client";
import { generateRequestId } from "../utils/requestId";
import type { ApiError } from "./types";
import { ApiRequestError } from "./types";

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || "";

const MAX_RETRIES = 2;
const RETRY_STATUS_CODES = [429, 503];

async function getAccessToken(): Promise<string | null> {
  const { data } = await supabase.auth.getSession();
  return data.session?.access_token ?? null;
}

function delay(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

export async function apiClient<T>(
  method: string,
  path: string,
  body?: unknown,
): Promise<T> {
  const token = await getAccessToken();
  const requestId = generateRequestId();

  let lastError: Error | null = null;

  for (let attempt = 0; attempt <= MAX_RETRIES; attempt++) {
    if (attempt > 0) {
      await delay(Math.pow(2, attempt) * 1000);
    }

    const headers: Record<string, string> = {
      "Content-Type": "application/json",
      "X-Request-Id": requestId,
    };
    if (token) {
      headers["Authorization"] = `Bearer ${token}`;
    }

    try {
      const response = await fetch(`${API_BASE_URL}${path}`, {
        method,
        headers,
        body: body ? JSON.stringify(body) : undefined,
      });

      const responseRequestId = response.headers.get("X-Request-Id");

      if (!response.ok) {
        const errorBody: ApiError = await response.json().catch(() => ({
          status: "error" as const,
          code: "UNKNOWN_ERROR",
          message: "エラーが発生しました",
          requestId: responseRequestId ?? requestId,
        }));
        errorBody.requestId = responseRequestId ?? requestId;

        if (
          RETRY_STATUS_CODES.includes(response.status) &&
          attempt < MAX_RETRIES
        ) {
          lastError = new ApiRequestError(response.status, errorBody);
          continue;
        }

        throw new ApiRequestError(response.status, errorBody);
      }

      return (await response.json()) as T;
    } catch (err) {
      if (err instanceof ApiRequestError) throw err;
      if (attempt < MAX_RETRIES && err instanceof TypeError) {
        lastError = err;
        continue;
      }
      throw err;
    }
  }

  throw lastError ?? new Error("Request failed after retries");
}

export const api = {
  get: <T>(path: string) => apiClient<T>("GET", path),
  post: <T>(path: string, body?: unknown) => apiClient<T>("POST", path, body),
  put: <T>(path: string, body?: unknown) => apiClient<T>("PUT", path, body),
  delete: <T>(path: string) => apiClient<T>("DELETE", path),
};
