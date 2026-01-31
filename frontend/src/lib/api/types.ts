/**
 * Shared API types
 *
 * Common types used across multiple API modules.
 * Uses Zod for runtime validation and type inference.
 */

import { z } from "zod";

export const UserSchema = z.object({
  id: z.number(),
  uuid: z.string(),
  username: z.string(),
  admin: z.boolean(),
});

export type User = z.infer<typeof UserSchema>;

export const SessionSchema = z.object({
  uuid: z.string(),
  createdAt: z.string(),
  userAgent: z.string(),
  ipAddress: z.string(),
  active: z.boolean(),
  revokedAt: z.string().nullable(),
});

export type Session = z.infer<typeof SessionSchema>;

export const PaginatedResponseSchema = <T extends z.ZodTypeAny>(
  itemSchema: T,
) =>
  z.object({
    offset: z.number(),
    total: z.number(),
    items: z.array(itemSchema),
  });

export type PaginatedResponse<T> = {
  offset: number;
  total: number;
  items: T[];
};

export const ApiErrorSchema = z.object({
  error: z.string(), // Error code to uniquely identify an error
  message: z.string(), // Error message to display to user in English
  key: z.string().optional(), // Optional key identifying the field causing the issue
});

export type ApiErrorType = z.infer<typeof ApiErrorSchema>;

export const ErrorResponseSchema = z.object({
  errors: z.array(ApiErrorSchema),
});

export type ErrorResponse = z.infer<typeof ErrorResponseSchema>;
