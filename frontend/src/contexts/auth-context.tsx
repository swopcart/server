import React, { createContext, useContext, useState, useEffect } from "react";
import { jwtDecode } from "jwt-decode";
import type { User } from "@/lib/api/types";
import type { LoginRequest } from "@/lib/api/auth";
import {
  login as apiLogin,
  logout as apiLogout,
  refreshToken as apiRefreshToken,
  getAccessToken,
  getRefreshToken,
  clearTokens,
} from "@/lib/api/auth";
import { ApiError } from "@/lib/api/client";

interface AuthState {
  user: User | null;
  sessionId: string | null;
  isLoading: boolean;
  isAuthenticated: boolean;
}

interface AuthContextValue extends AuthState {
  login: (req: LoginRequest) => Promise<void>;
  logout: () => void;
}

const AuthContext = createContext<AuthContextValue | undefined>(undefined);

interface AccessTokenPayload {
  sub: string;
  sid: string;
  exp: number;
  username: string;
  admin: boolean;
}

interface TokenData {
  user: User;
  sessionId: string;
}

function getDataFromToken(accessToken: string): TokenData | null {
  try {
    const payload = jwtDecode<AccessTokenPayload>(accessToken);

    // Check expiration
    if (payload.exp * 1000 < Date.now()) {
      return null;
    }

    return {
      user: {
        id: 0, // Not available in token
        uuid: payload.sub,
        username: payload.username,
        admin: payload.admin,
      },
      sessionId: payload.sid,
    };
  } catch {
    return null;
  }
}

function getInitialAuthState(): AuthState {
  const accessToken = getAccessToken();
  const refreshTokenValue = getRefreshToken();

  if (accessToken) {
    const tokenData = getDataFromToken(accessToken);
    if (tokenData) {
      return {
        user: tokenData.user,
        sessionId: tokenData.sessionId,
        isLoading: false,
        isAuthenticated: true,
      };
    }
  }

  // Access token is missing or expired - if we have a refresh token,
  // start in loading state so we can attempt refresh on mount
  if (refreshTokenValue) {
    return {
      user: null,
      sessionId: null,
      isLoading: true,
      isAuthenticated: false,
    };
  }

  // No tokens at all
  clearTokens();
  return {
    user: null,
    sessionId: null,
    isLoading: false,
    isAuthenticated: false,
  };
}

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [state, setState] = useState<AuthState>(getInitialAuthState);

  // Attempt to refresh token on mount if we started in loading state
  useEffect(() => {
    const needsRefresh =
      getRefreshToken() && !getDataFromToken(getAccessToken() || "");
    if (!needsRefresh) {
      return;
    }

    apiRefreshToken().then((newAccessToken) => {
      if (newAccessToken) {
        const tokenData = getDataFromToken(newAccessToken);
        if (tokenData) {
          setState({
            user: tokenData.user,
            sessionId: tokenData.sessionId,
            isLoading: false,
            isAuthenticated: true,
          });
          return;
        }
      }

      // Refresh failed
      clearTokens();
      setState({
        user: null,
        sessionId: null,
        isLoading: false,
        isAuthenticated: false,
      });
    });
  }, []);

  const login = async (req: LoginRequest): Promise<void> => {
    const response = await apiLogin(req);
    const tokenData = getDataFromToken(getAccessToken() || "");
    setState({
      user: response.user,
      sessionId: tokenData?.sessionId ?? null,
      isLoading: false,
      isAuthenticated: true,
    });
  };

  const logout = (): void => {
    // Clear local state immediately for instant UI feedback
    clearTokens();
    setState({
      user: null,
      sessionId: null,
      isLoading: false,
      isAuthenticated: false,
    });

    // Invalidate session on server in the background
    apiLogout().catch(() => {
      // Server-side logout failed, but local state is already cleared
    });
  };

  return (
    <AuthContext.Provider value={{ ...state, login, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth(): AuthContextValue {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return context;
}

export { ApiError };
