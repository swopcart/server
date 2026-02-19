/**
 * Games API
 *
 * Handles game discovery and retrieval, including listing, searching, and downloading.
 * Endpoints under /api/v0/games/* and /api/v0/libraries/:libraryId/games
 */

import { z } from "zod";
import { ApiError, fetchWithAuth } from "./client";
import {
  GameSchema,
  GameListResponseSchema,
  type Game,
  type GameListResponse,
} from "./types";

// Update game metadata request
export const UpdateGameMetadataRequestSchema = z.object({
  title: z.string().optional(),
  developer: z.string().optional(),
  publisher: z.string().optional(),
  description: z.string().optional(),
  externalIds: z.record(z.string()).optional(),
  regions: z.array(z.string()).optional(),
  tags: z.array(z.string()).optional(),
});

export type UpdateGameMetadataRequest = z.infer<
  typeof UpdateGameMetadataRequestSchema
>;

// API functions

/**
 * List games in a library with pagination and search
 */
export async function listGamesByLibrary(
  libraryId: string,
  offset?: number,
  limit?: number,
  search?: string,
): Promise<GameListResponse> {
  const params = new URLSearchParams();

  if (offset !== undefined) {
    params.append("offset", offset.toString());
  }
  if (limit !== undefined) {
    params.append("limit", limit.toString());
  }
  if (search) {
    params.append("search", search);
  }

  const path =
    `/games/by-library/${libraryId}` +
    (params.toString() ? `?${params.toString()}` : "");

  const response = await fetchWithAuth(path, {
    method: "GET",
  });

  if (!response.ok) {
    const body = await response.json();
    throw new ApiError(response.status, body);
  }

  return GameListResponseSchema.parse(await response.json());
}

/**
 * Get a single game by ID with all versions and metadata
 */
export async function getGame(gameId: string): Promise<Game> {
  const response = await fetchWithAuth(`/games/${gameId}`, {
    method: "GET",
  });

  if (!response.ok) {
    const body = await response.json();
    throw new ApiError(response.status, body);
  }

  return GameSchema.parse(await response.json());
}

/**
 * Update game metadata (admin-only)
 */
export async function updateGameMetadata(
  gameId: string,
  request: UpdateGameMetadataRequest,
): Promise<Game> {
  const response = await fetchWithAuth(`/games/${gameId}`, {
    method: "PATCH",
    body: JSON.stringify(request),
  });

  if (!response.ok) {
    const body = await response.json();
    throw new ApiError(response.status, body);
  }

  return GameSchema.parse(await response.json());
}

/**
 * Download a specific game version
 * Returns a blob that can be used to create a download link
 */
export async function downloadGameVersion(
  gameId: string,
  versionId: string,
): Promise<Blob> {
  const response = await fetchWithAuth(
    `/games/${gameId}/versions/${versionId}/download`,
    {
      method: "GET",
    },
  );

  if (!response.ok) {
    const body = await response.json();
    throw new ApiError(response.status, body);
  }

  return response.blob();
}
