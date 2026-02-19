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

// Platform schemas
export const PlatformSchema = z.object({
  id: z.number(),
  name: z.string(),
  description: z.string(),
  extensions: z.array(z.string()),
});

export type Platform = z.infer<typeof PlatformSchema>;

// Game metadata schemas
export const ExternalIdsSchema = z.record(z.string());

export const GameMetadataSchema = z.object({
  title: z.string(),
  platform: z.string().optional(),
  developer: z.string().optional(),
  publisher: z.string().optional(),
  releaseDate: z.string().optional(),
  description: z.string().optional(),
  externalIds: ExternalIdsSchema.optional(),
  regions: z.array(z.string()).optional(),
  tags: z.array(z.string()).optional(),
});

export type GameMetadata = z.infer<typeof GameMetadataSchema>;

// Game version schemas
export const GameVersionSchema = z.object({
  id: z.string(),
  versionName: z.string(),
  filePath: z.string(),
  fileSize: z.number(),
  md5: z.string().optional(),
  sha1: z.string().optional(),
  sha256: z.string().optional(),
  blake3: z.string().optional(),
  metadataPath: z.string().optional(),
  metadataJson: GameMetadataSchema.optional(),
});

export type GameVersion = z.infer<typeof GameVersionSchema>;

// Game schemas
export const GameSchema = z.object({
  id: z.string(),
  libraryId: z.string(),
  title: z.string(),
  platformId: z.number(),
  developer: z.string().optional(),
  publisher: z.string().optional(),
  releasedDate: z.string().nullable().optional(),
  description: z.string().optional(),
  versions: z.array(GameVersionSchema).optional(),
});

export type Game = z.infer<typeof GameSchema>;

// Library schemas
export const LibrarySchema = z.object({
  id: z.string(),
  name: z.string(),
  description: z.string(),
  platformId: z.number(),
  platformName: z.string().optional(),
  paths: z.array(z.string()),
  lastScannedAt: z.string().nullable().optional(),
  scanStatus: z.string(), // "idle", "scanning", "error"
  lastScanError: z.string().nullable().optional(),
  currentScanJobId: z.string().nullable().optional(),
  gameCount: z.number().optional(),
});

export type Library = z.infer<typeof LibrarySchema>;

// List response schemas
export const LibraryListResponseSchema = z.object({
  items: z.array(LibrarySchema),
  total: z.number(),
});

export type LibraryListResponse = z.infer<typeof LibraryListResponseSchema>;

export const GameListResponseSchema = z.object({
  items: z.array(GameSchema),
  offset: z.number(),
  total: z.number(),
});

export type GameListResponse = z.infer<typeof GameListResponseSchema>;
