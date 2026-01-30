const API_BASE = "/api/v0";

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

// Types
export interface User {
  id: number;
  uuid: string;
  username: string;
  admin: boolean;
}

export interface LoginRequest {
  username: string;
  password: string;
  totp?: string;
}

export interface LoginResponse {
  refreshToken: string;
  accessToken: string;
  user: User;
}

export interface LoginErrorResponse {
  error: string;
  requiresTotp?: boolean;
}

export interface RefreshResponse {
  accessToken: string;
}

export interface Session {
  uuid: string;
  createdAt: string;
  userAgent: string;
  ipAddress: string;
  active: boolean;
  revokedAt: string | null;
}

export interface PaginatedResponse<T> {
  offset: number;
  total: number;
  items: T[];
}

// API Error class
export class ApiError extends Error {
  status: number;
  body: unknown;

  constructor(status: number, body: unknown) {
    super(
      typeof body === "object" && body && "error" in body
        ? String((body as { error: string }).error)
        : "API Error",
    );
    this.status = status;
    this.body = body;
  }

  get requiresTotp(): boolean {
    return (
      typeof this.body === "object" &&
      this.body !== null &&
      "requiresTotp" in this.body &&
      (this.body as { requiresTotp?: boolean }).requiresTotp === true
    );
  }
}

// Fetch wrapper with auth
async function fetchWithAuth(
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

// Exported refresh function for auth context initialization
export async function refreshToken(): Promise<string | null> {
  const token = getRefreshToken();
  if (!token) {
    return null;
  }

  const success = await refreshAccessToken(token);
  if (success) {
    return getAccessToken();
  }
  return null;
}

// Auth API functions
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

  const data = body as LoginResponse;
  setTokens(data.accessToken, data.refreshToken);
  return data;
}

export async function logout(): Promise<void> {
  await fetchWithAuth("/auth/", { method: "DELETE" });
  clearTokens();
}

export async function listSessions(): Promise<PaginatedResponse<Session>> {
  const response = await fetchWithAuth("/auth/sessions/", {
    method: "GET",
  });

  if (!response.ok) {
    const body = await response.json();
    throw new ApiError(response.status, body);
  }

  return response.json() as Promise<PaginatedResponse<Session>>;
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

// User API functions
export interface UserDetails {
  uuid: string;
  username: string;
  admin: boolean;
  totpEnabled: boolean;
}

export async function getUserDetails(userUuid: string): Promise<UserDetails> {
  const response = await fetchWithAuth(`/users/${userUuid}`, {
    method: "GET",
  });

  if (!response.ok) {
    const body = await response.json();
    throw new ApiError(response.status, body);
  }

  return response.json() as Promise<UserDetails>;
}

export async function changePassword(
  userUuid: string,
  currentPassword: string,
  newPassword: string,
): Promise<void> {
  const response = await fetchWithAuth(`/users/${userUuid}/password`, {
    method: "POST",
    body: JSON.stringify({ currentPassword, newPassword }),
  });

  if (!response.ok) {
    const body = await response.json();
    throw new ApiError(response.status, body);
  }
}

// TOTP API functions
export interface GenerateTOTPResponse {
  secret: string;
  url: string;
}

export async function generateTOTP(
  userUuid: string,
  password: string,
): Promise<GenerateTOTPResponse> {
  const response = await fetchWithAuth(`/users/${userUuid}/totp/generate`, {
    method: "POST",
    body: JSON.stringify({ password }),
  });

  if (!response.ok) {
    const body = await response.json();
    throw new ApiError(response.status, body);
  }

  return response.json() as Promise<GenerateTOTPResponse>;
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
  password: string,
): Promise<void> {
  const response = await fetchWithAuth(`/users/${userUuid}/totp`, {
    method: "DELETE",
    body: JSON.stringify({ password }),
  });

  if (!response.ok) {
    const body = await response.json();
    throw new ApiError(response.status, body);
  }
}

// User management API functions
export interface UserListItem extends UserDetails {
  createdAt: string;
}

export async function listUsers(): Promise<PaginatedResponse<UserListItem>> {
  const response = await fetchWithAuth("/users/", {
    method: "GET",
  });

  if (!response.ok) {
    const body = await response.json();
    throw new ApiError(response.status, body);
  }

  return response.json() as Promise<PaginatedResponse<UserListItem>>;
}

export async function adminChangePassword(
  userUuid: string,
  newPassword: string,
): Promise<void> {
  const response = await fetchWithAuth(`/users/${userUuid}/password`, {
    method: "POST",
    body: JSON.stringify({ newPassword }),
  });

  if (!response.ok) {
    const body = await response.json();
    throw new ApiError(response.status, body);
  }
}

export async function adminRemoveTotp(userUuid: string): Promise<void> {
  const response = await fetchWithAuth(`/users/${userUuid}/totp`, {
    method: "DELETE",
  });

  if (!response.ok) {
    const body = await response.json();
    throw new ApiError(response.status, body);
  }
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

export async function listUserSessions(
  userUuid: string,
): Promise<PaginatedResponse<Session>> {
  const response = await fetchWithAuth(`/users/${userUuid}/sessions/`, {
    method: "GET",
  });

  if (!response.ok) {
    const body = await response.json();
    throw new ApiError(response.status, body);
  }

  return response.json() as Promise<PaginatedResponse<Session>>;
}

export async function adminRevokeSession(
  userUuid: string,
  sessionUuid: string,
): Promise<void> {
  const response = await fetchWithAuth(
    `/users/${userUuid}/sessions/${sessionUuid}`,
    {
      method: "DELETE",
    },
  );

  if (!response.ok) {
    const body = await response.json();
    throw new ApiError(response.status, body);
  }
}

export interface CreateUserRequest {
  username: string;
  password: string;
  admin: boolean;
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

  return response.json() as Promise<UserListItem>;
}
