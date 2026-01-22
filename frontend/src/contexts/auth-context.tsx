import React, { createContext, useContext, useState, useEffect } from "react";
import { jwtDecode } from "jwt-decode";
import type { User, LoginRequest } from "@/lib/api";
import {
  login as apiLogin,
  logout as apiLogout,
  refreshToken as apiRefreshToken,
  getAccessToken,
  getRefreshToken,
  clearTokens,
  ApiError,
} from "@/lib/api";

interface AuthState {
  user: User | null;
  isLoading: boolean;
  isAuthenticated: boolean;
}

interface AuthContextValue extends AuthState {
  login: (req: LoginRequest) => Promise<void>;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | undefined>(undefined);

interface AccessTokenPayload {
  sub: string;
  sid: string;
  exp: number;
  username: string;
  admin: boolean;
}

function getUserFromToken(accessToken: string): User | null {
  try {
    const payload = jwtDecode<AccessTokenPayload>(accessToken);

    // Check expiration
    if (payload.exp * 1000 < Date.now()) {
      return null;
    }

    return {
      id: 0, // Not available in token
      uuid: payload.sub,
      username: payload.username,
      admin: payload.admin,
    };
  } catch {
    return null;
  }
}

function getInitialAuthState(): AuthState {
  const accessToken = getAccessToken();
  const refreshTokenValue = getRefreshToken();

  if (accessToken) {
    const user = getUserFromToken(accessToken);
    if (user) {
      return {
        user,
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
      isLoading: true,
      isAuthenticated: false,
    };
  }

  // No tokens at all
  clearTokens();
  return {
    user: null,
    isLoading: false,
    isAuthenticated: false,
  };
}

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [state, setState] = useState<AuthState>(getInitialAuthState);

  // Attempt to refresh token on mount if we started in loading state
  useEffect(() => {
    const needsRefresh =
      getRefreshToken() && !getUserFromToken(getAccessToken() || "");
    if (!needsRefresh) {
      return;
    }

    apiRefreshToken().then((newAccessToken) => {
      if (newAccessToken) {
        const user = getUserFromToken(newAccessToken);
        if (user) {
          setState({
            user,
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
        isLoading: false,
        isAuthenticated: false,
      });
    });
  }, []);

  const login = async (req: LoginRequest): Promise<void> => {
    const response = await apiLogin(req);
    setState({
      user: response.user,
      isLoading: false,
      isAuthenticated: true,
    });
  };

  const logout = async (): Promise<void> => {
    try {
      await apiLogout();
    } catch {
      // Even if API call fails, clear local state
    }
    clearTokens();
    setState({
      user: null,
      isLoading: false,
      isAuthenticated: false,
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
