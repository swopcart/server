/**
 * HTTP client infrastructure
 *
 * Provides the base HTTP client with authentication, token refresh,
 * and error handling for all API requests.
 */

import { ErrorResponseSchema, type ErrorResponse } from "./types";

export const API_BASE = "/api/v0";

// Storage keys
const ACCESS_TOKEN_KEY = "swopcart-access-token";
const REFRESH_TOKEN_KEY = "swopcart-refresh-token";

// Token management
export function getAccessToken(): string | null {
  return localStorage.getItem(ACCESS_TOKEN_KEY);
}

export function getRefreshToken(): string | null {
  return localStorage.getItem(REFRESH_TOKEN_KEY);
}

export function setTokens(accessToken: string, refreshToken: string): void {
  localStorage.setItem(ACCESS_TOKEN_KEY, accessToken);
  localStorage.setItem(REFRESH_TOKEN_KEY, refreshToken);
}

export function clearTokens(): void {
  localStorage.removeItem(ACCESS_TOKEN_KEY);
  localStorage.removeItem(REFRESH_TOKEN_KEY);
}

// API Error class
export class ApiError extends Error {
  status: number;
  body: unknown;
  errorResponse: ErrorResponse | null = null;

  constructor(status: number, body: unknown) {
    // Try to parse as ErrorResponse
    let errorResponse: ErrorResponse | null = null;
    try {
      errorResponse = ErrorResponseSchema.parse(body);
    } catch {
      // Not a valid ErrorResponse, ignore
    }

    // Use first error message if available, otherwise generic message
    const message = errorResponse?.errors[0]?.message ?? "API Error";

    super(message);
    this.status = status;
    this.body = body;
    this.errorResponse = errorResponse;
  }

  /**
   * Check if this error requires TOTP verification
   * Returns true if there's a "RequiresTOTP" error with key ".totp"
   */
  get requiresTotp(): boolean {
    return (
      this.errorResponse?.errors.some(
        (err) => err.error === "RequiresTOTP" && err.key === ".totp",
      ) ?? false
    );
  }

  /**
   * Get all error messages from the response
   */
  getMessages(): string[] {
    return this.errorResponse?.errors.map((err) => err.message) ?? [];
  }

  /**
   * Get errors for a specific field key
   */
  getErrorsForKey(key: string): string[] {
    return (
      this.errorResponse?.errors
        .filter((err) => err.key === key)
        .map((err) => err.message) ?? []
    );
  }
}

// Internal token refresh
interface RefreshResponse {
  accessToken: string;
}

async function refreshAccessToken(token: string): Promise<boolean> {
  try {
    const response = await fetch(`${API_BASE}/auth/refresh`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ refreshToken: token }),
    });

    if (!response.ok) {
      return false;
    }

    const data = (await response.json()) as RefreshResponse;
    localStorage.setItem(ACCESS_TOKEN_KEY, data.accessToken);
    return true;
  } catch {
    return false;
  }
}

// Fetch wrapper with auth and auto-refresh
export async function fetchWithAuth(
  path: string,
  options: RequestInit = {},
  retry = true,
): Promise<Response> {
  const accessToken = getAccessToken();

  const headers = new Headers(options.headers);
  if (accessToken) {
    headers.set("Authorization", `Bearer ${accessToken}`);
  }
  if (options.body && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  const response = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers,
  });

  // Handle 401 - try refresh
  if (response.status === 401 && retry) {
    const refreshToken = getRefreshToken();
    if (refreshToken) {
      const refreshed = await refreshAccessToken(refreshToken);
      if (refreshed) {
        return fetchWithAuth(path, options, false);
      }
    }
    clearTokens();
  }

  return response;
}
