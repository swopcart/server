/**
 * Jobs API
 *
 * Handles background job management, job triggering, and execution history.
 * Endpoints under /api/v0/jobs/*
 */

import { z } from "zod";
import { ApiError, fetchWithAuth } from "./client";

// Job schemas
export const JobSchema = z.object({
  uuid: z.string(),
  name: z.string(),
  description: z.string(),
  schedule: z.string().nullable(),
  enabled: z.boolean(),
  priority: z.number(),
  queue: z.string(),
  defaultParameters: z.string().nullable(),
  lastRunAt: z.string().nullable(),
  nextRunAt: z.string().nullable(),
});

export type Job = z.infer<typeof JobSchema>;

// Job execution schemas
export const JobExecutionSchema = z.object({
  uuid: z.string(),
  jobId: z.number(),
  parameters: z.string().nullable(),
  startedAt: z.string(),
  completedAt: z.string().nullable(),
  duration: z.number().nullable(),
  status: z.string(), // "pending", "running", "completed", "failed"
  output: z.string().nullable(),
  error: z.string().nullable(),
  triggerType: z.string(),
  recordsComplete: z.number(),
  recordsTotal: z.number(),
  currentRecord: z.string().nullable(),
});

export type JobExecution = z.infer<typeof JobExecutionSchema>;

// Trigger response
export const TriggerJobResponseSchema = z.object({
  executionUuid: z.string(),
  status: z.string(),
});

export type TriggerJobResponse = z.infer<typeof TriggerJobResponseSchema>;

// API functions
export async function listJobs(): Promise<Job[]> {
  const response = await fetchWithAuth("/jobs/", {
    method: "GET",
  });

  if (!response.ok) {
    const body = await response.json();
    throw new ApiError(response.status, body);
  }

  return z.array(JobSchema).parse(await response.json());
}

export async function getJobDetails(jobName: string): Promise<Job> {
  const response = await fetchWithAuth(`/jobs/${jobName}`, {
    method: "GET",
  });

  if (!response.ok) {
    const body = await response.json();
    throw new ApiError(response.status, body);
  }

  return JobSchema.parse(await response.json());
}

export async function triggerJob(
  jobName: string,
  parameters?: Record<string, string>,
): Promise<TriggerJobResponse> {
  const response = await fetchWithAuth(`/jobs/${jobName}/trigger`, {
    method: "POST",
    body: parameters ? JSON.stringify({ parameters }) : JSON.stringify({}),
  });

  if (!response.ok) {
    const body = await response.json();
    throw new ApiError(response.status, body);
  }

  return TriggerJobResponseSchema.parse(await response.json());
}

export async function getJobHistory(
  jobName: string,
  limit?: number,
): Promise<JobExecution[]> {
  const params = new URLSearchParams();
  if (limit) {
    params.append("limit", limit.toString());
  }

  const path = `/jobs/${jobName}/history${params.toString() ? `?${params.toString()}` : ""}`;

  const response = await fetchWithAuth(path, {
    method: "GET",
  });

  if (!response.ok) {
    const body = await response.json();
    throw new ApiError(response.status, body);
  }

  return z.array(JobExecutionSchema).parse(await response.json());
}

export async function listRecentExecutions(
  limit?: number,
): Promise<JobExecution[]> {
  const params = new URLSearchParams();
  if (limit) {
    params.append("limit", limit.toString());
  }

  const path = `/jobs/executions/recent${params.toString() ? `?${params.toString()}` : ""}`;

  const response = await fetchWithAuth(path, {
    method: "GET",
  });

  if (!response.ok) {
    const body = await response.json();
    throw new ApiError(response.status, body);
  }

  return z.array(JobExecutionSchema).parse(await response.json());
}

export async function getExecution(
  executionUuid: string,
): Promise<JobExecution> {
  const response = await fetchWithAuth(`/jobs/executions/${executionUuid}`, {
    method: "GET",
  });

  if (!response.ok) {
    const body = await response.json();
    throw new ApiError(response.status, body);
  }

  return JobExecutionSchema.parse(await response.json());
}

export async function updateJobSettings(
  jobName: string,
  settings: { enabled?: boolean },
): Promise<void> {
  const response = await fetchWithAuth(`/jobs/${jobName}`, {
    method: "PATCH",
    body: JSON.stringify(settings),
  });

  if (!response.ok) {
    const body = await response.json();
    throw new ApiError(response.status, body);
  }
}
