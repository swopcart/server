/**
 * Platforms API
 *
 * Handles platform enumeration for library management.
 * Endpoints under /api/v0/platforms/*
 */

import { z } from "zod";
import { ApiError, fetchWithAuth } from "./client";
import { PlatformSchema, type Platform } from "./types";

/**
 * List all available platforms
 */
export async function listPlatforms(): Promise<Platform[]> {
  const response = await fetchWithAuth("/platforms/", {
    method: "GET",
  });

  if (!response.ok) {
    const body = await response.json();
    throw new ApiError(response.status, body);
  }

  const data = await response.json();
  return z.array(PlatformSchema).parse(data.items);
}
