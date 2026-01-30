/**
 * Authentication API
 *
 * Handles authentication, token management, and session management.
 * Endpoints under /api/v0/auth/*
 */

import { z } from "zod";
import {
  ApiError,
  fetchWithAuth,
  getAccessToken,
  getRefreshToken,
  setTokens,
  clearTokens,
  API_BASE,
} from "./client";
import { UserSchema, SessionSchema, PaginatedResponseSchema } from "./types";
import type { Session, PaginatedResponse } from "./types";

// Auth request/response schemas
export const LoginRequestSchema = z.object({
  username: z.string(),
  password: z.string(),
  totp: z.string().optional(),
});

export type LoginRequest = z.infer<typeof LoginRequestSchema>;

export const LoginResponseSchema = z.object({
  refreshToken: z.string(),
  accessToken: z.string(),
  user: UserSchema,
});

export type LoginResponse = z.infer<typeof LoginResponseSchema>;

// Auth functions
export async function login(req: LoginRequest): Promise<LoginResponse> {
  const response = await fetch(`${API_BASE}/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(req),
  });

  const body = await response.json();

  if (!response.ok) {
    throw new ApiError(response.status, body);
  }

  const data = LoginResponseSchema.parse(body);
  setTokens(data.accessToken, data.refreshToken);
  return data;
}

export async function logout(): Promise<void> {
  await fetchWithAuth("/auth/", { method: "DELETE" });
  clearTokens();
}

export async function refreshToken(): Promise<string | null> {
  const token = getRefreshToken();
  if (!token) {
    return null;
  }

  try {
    const response = await fetch(`${API_BASE}/auth/refresh`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ refreshToken: token }),
    });

    if (!response.ok) {
      return null;
    }

    const RefreshResponseSchema = z.object({
      accessToken: z.string(),
    });

    const data = RefreshResponseSchema.parse(await response.json());
    const currentRefreshToken = getRefreshToken();
    if (currentRefreshToken) {
      setTokens(data.accessToken, currentRefreshToken);
    }
    return data.accessToken;
  } catch {
    return null;
  }
}

// Session management
export async function listSessions(): Promise<PaginatedResponse<Session>> {
  const response = await fetchWithAuth("/auth/sessions/", {
    method: "GET",
  });

  if (!response.ok) {
    const body = await response.json();
    throw new ApiError(response.status, body);
  }

  const schema = PaginatedResponseSchema(SessionSchema);
  return schema.parse(await response.json());
}

export async function revokeSession(sessionUuid: string): Promise<void> {
  const response = await fetchWithAuth(`/auth/sessions/${sessionUuid}`, {
    method: "DELETE",
  });

  if (!response.ok) {
    const body = await response.json();
    throw new ApiError(response.status, body);
  }
}

// Re-export token management functions for convenience
export { getAccessToken, getRefreshToken, clearTokens };
