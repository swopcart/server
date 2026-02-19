/**
 * Libraries API
 *
 * Handles library management (admin-only), including CRUD operations and scan triggering.
 * Endpoints under /api/v0/libraries/*
 */

import { z } from "zod";
import { ApiError, fetchWithAuth } from "./client";
import { LibrarySchema, type Library } from "./types";

// Re-export types for external use
export type { Library };

// Create library request
export const CreateLibraryRequestSchema = z.object({
  name: z.string().min(1),
  description: z.string().optional(),
  platformId: z.number(),
  paths: z.array(z.string()).min(1),
});

export type CreateLibraryRequest = z.infer<typeof CreateLibraryRequestSchema>;

// Update library request
export const UpdateLibraryRequestSchema = z.object({
  name: z.string().optional(),
  description: z.string().optional(),
  paths: z.array(z.string()).optional(),
});

export type UpdateLibraryRequest = z.infer<typeof UpdateLibraryRequestSchema>;

// Scan response
export const TriggerScanResponseSchema = z.object({
  executionUuid: z.string(),
  status: z.string(),
});

export type TriggerScanResponse = z.infer<typeof TriggerScanResponseSchema>;

// API functions

/**
 * List all libraries (admin-only)
 */
export async function listLibraries(): Promise<Library[]> {
  const response = await fetchWithAuth("/libraries/", {
    method: "GET",
  });

  if (!response.ok) {
    const body = await response.json();
    throw new ApiError(response.status, body);
  }

  const data = await response.json();
  return z.array(LibrarySchema).parse(data.items);
}

/**
 * Get a single library by ID (admin-only)
 */
export async function getLibrary(libraryId: string): Promise<Library> {
  const response = await fetchWithAuth(`/libraries/${libraryId}`, {
    method: "GET",
  });

  if (!response.ok) {
    const body = await response.json();
    throw new ApiError(response.status, body);
  }

  return LibrarySchema.parse(await response.json());
}

/**
 * Create a new library (admin-only)
 */
export async function createLibrary(
  request: CreateLibraryRequest,
): Promise<Library> {
  const response = await fetchWithAuth("/libraries/", {
    method: "POST",
    body: JSON.stringify({
      name: request.name,
      description: request.description || "",
      platformId: request.platformId,
      paths: request.paths,
    }),
  });

  if (!response.ok) {
    const body = await response.json();
    throw new ApiError(response.status, body);
  }

  return LibrarySchema.parse(await response.json());
}

/**
 * Update a library (admin-only)
 */
export async function updateLibrary(
  libraryId: string,
  request: UpdateLibraryRequest,
): Promise<Library> {
  const response = await fetchWithAuth(`/libraries/${libraryId}`, {
    method: "PATCH",
    body: JSON.stringify(request),
  });

  if (!response.ok) {
    const body = await response.json();
    throw new ApiError(response.status, body);
  }

  return LibrarySchema.parse(await response.json());
}

/**
 * Delete a library (admin-only)
 */
export async function deleteLibrary(libraryId: string): Promise<void> {
  const response = await fetchWithAuth(`/libraries/${libraryId}`, {
    method: "DELETE",
  });

  if (!response.ok) {
    const body = await response.json();
    throw new ApiError(response.status, body);
  }
}

/**
 * Trigger a library scan (admin-only)
 */
export async function triggerLibraryScan(
  libraryId: string,
): Promise<TriggerScanResponse> {
  const response = await fetchWithAuth(`/libraries/${libraryId}/scan`, {
    method: "POST",
    body: JSON.stringify({}),
  });

  if (!response.ok) {
    const body = await response.json();
    throw new ApiError(response.status, body);
  }

  return TriggerScanResponseSchema.parse(await response.json());
}
