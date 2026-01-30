/**
 * Users API
 *
 * Handles user management, password changes, TOTP configuration,
 * and user session management.
 * Endpoints under /api/v0/users/*
 */

import { z } from "zod";
import { ApiError, fetchWithAuth } from "./client";
import { SessionSchema, PaginatedResponseSchema } from "./types";
import type { Session, PaginatedResponse } from "./types";

// User detail schemas
export const UserDetailsSchema = z.object({
  uuid: z.string(),
  username: z.string(),
  admin: z.boolean(),
  totpEnabled: z.boolean(),
});

export type UserDetails = z.infer<typeof UserDetailsSchema>;

export const UserListItemSchema = UserDetailsSchema.extend({
  createdAt: z.string(),
});

export type UserListItem = z.infer<typeof UserListItemSchema>;

export const CreateUserRequestSchema = z.object({
  username: z.string(),
  password: z.string(),
  admin: z.boolean(),
});

export type CreateUserRequest = z.infer<typeof CreateUserRequestSchema>;

// TOTP schemas
export const GenerateTOTPResponseSchema = z.object({
  secret: z.string(),
  url: z.string(),
});

export type GenerateTOTPResponse = z.infer<typeof GenerateTOTPResponseSchema>;

// User management functions
export async function getUserDetails(userUuid: string): Promise<UserDetails> {
  const response = await fetchWithAuth(`/users/${userUuid}`, {
    method: "GET",
  });

  if (!response.ok) {
    const body = await response.json();
    throw new ApiError(response.status, body);
  }

  return UserDetailsSchema.parse(await response.json());
}

export async function listUsers(): Promise<PaginatedResponse<UserListItem>> {
  const response = await fetchWithAuth("/users/", {
    method: "GET",
  });

  if (!response.ok) {
    const body = await response.json();
    throw new ApiError(response.status, body);
  }

  const schema = PaginatedResponseSchema(UserListItemSchema);
  return schema.parse(await response.json());
}

export async function createUser(
  req: CreateUserRequest,
): Promise<UserListItem> {
  const response = await fetchWithAuth("/users/", {
    method: "POST",
    body: JSON.stringify(req),
  });

  if (!response.ok) {
    const body = await response.json();
    throw new ApiError(response.status, body);
  }

  return UserListItemSchema.parse(await response.json());
}

export async function deleteUser(userUuid: string): Promise<void> {
  const response = await fetchWithAuth(`/users/${userUuid}`, {
    method: "DELETE",
  });

  if (!response.ok) {
    const body = await response.json();
    throw new ApiError(response.status, body);
  }
}

// Password management
export async function changePassword(
  userUuid: string,
  newPassword: string,
  currentPassword?: string,
): Promise<void> {
  const body: { newPassword: string; currentPassword?: string } = {
    newPassword,
  };
  if (currentPassword !== undefined) {
    body.currentPassword = currentPassword;
  }

  const response = await fetchWithAuth(`/users/${userUuid}/password`, {
    method: "POST",
    body: JSON.stringify(body),
  });

  if (!response.ok) {
    const responseBody = await response.json();
    throw new ApiError(response.status, responseBody);
  }
}

// TOTP management
export async function generateTOTP(
  userUuid: string,
  password?: string,
): Promise<GenerateTOTPResponse> {
  const body: { password?: string } = {};
  if (password !== undefined) {
    body.password = password;
  }

  const response = await fetchWithAuth(`/users/${userUuid}/totp/generate`, {
    method: "POST",
    body: JSON.stringify(body),
  });

  if (!response.ok) {
    const responseBody = await response.json();
    throw new ApiError(response.status, responseBody);
  }

  return GenerateTOTPResponseSchema.parse(await response.json());
}

export async function enableTOTP(
  userUuid: string,
  secret: string,
  token: string,
): Promise<void> {
  const response = await fetchWithAuth(`/users/${userUuid}/totp`, {
    method: "POST",
    body: JSON.stringify({ secret, token }),
  });

  if (!response.ok) {
    const body = await response.json();
    throw new ApiError(response.status, body);
  }
}

export async function disableTOTP(
  userUuid: string,
  password?: string,
): Promise<void> {
  const body: { password?: string } = {};
  if (password !== undefined) {
    body.password = password;
  }

  const response = await fetchWithAuth(`/users/${userUuid}/totp`, {
    method: "DELETE",
    body: JSON.stringify(body),
  });

  if (!response.ok) {
    const responseBody = await response.json();
    throw new ApiError(response.status, responseBody);
  }
}

// User session management
export async function listUserSessions(
  userUuid: string,
): Promise<PaginatedResponse<Session>> {
  const response = await fetchWithAuth(`/users/${userUuid}/sessions`, {
    method: "GET",
  });

  if (!response.ok) {
    const body = await response.json();
    throw new ApiError(response.status, body);
  }

  const schema = PaginatedResponseSchema(SessionSchema);
  return schema.parse(await response.json());
}
