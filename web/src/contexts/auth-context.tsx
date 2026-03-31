import { createContext, useContext, useEffect, useState } from "react";
import type { PropsWithChildren } from "react";
import type { LoginPayload, UserItem } from "../types/api";
import { getCurrentUser, login as loginRequest, logout as logoutRequest } from "../services/auth";
import { clearAuthStorage, getToken, getUser, setToken, setUser } from "../services/storage";

interface AuthContextValue {
  user: UserItem | null;
  token: string;
  loading: boolean;
  isAuthenticated: boolean;
  loginAction: (payload: LoginPayload) => Promise<void>;
  logoutAction: () => Promise<void>;
  refreshUser: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | undefined>(undefined);

export function AuthProvider({ children }: PropsWithChildren) {
  const [user, setUserState] = useState<UserItem | null>(() => getUser());
  const [token, setTokenState] = useState<string>(() => getToken());
  const [loading, setLoading] = useState<boolean>(() => Boolean(getToken()));

  useEffect(() => {
    if (!token) {
      setLoading(false);
      return;
    }

    let cancelled = false;

    async function bootstrap() {
      try {
        const profile = await getCurrentUser();
        if (cancelled) {
          return;
        }
        setUser(profile);
        setUserState(profile);
      } catch {
        if (!cancelled) {
          clearAuthStorage();
          setTokenState("");
          setUserState(null);
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    void bootstrap();

    return () => {
      cancelled = true;
    };
  }, [token]);

  async function loginAction(payload: LoginPayload) {
    const result = await loginRequest(payload);
    setToken(result.token);
    setUser(result.user);
    setTokenState(result.token);
    setUserState(result.user);
  }

  async function logoutAction() {
    try {
      if (token) {
        await logoutRequest();
      }
    } catch {
      // keep local cleanup even if the backend token has already expired
    } finally {
      clearAuthStorage();
      setTokenState("");
      setUserState(null);
    }
  }

  async function refreshUser() {
    if (!token) {
      return;
    }
    const profile = await getCurrentUser();
    setUser(profile);
    setUserState(profile);
  }

  return (
    <AuthContext.Provider
      value={{
        user,
        token,
        loading,
        isAuthenticated: Boolean(token),
        loginAction,
        logoutAction,
        refreshUser,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth must be used within AuthProvider");
  }
  return context;
}
