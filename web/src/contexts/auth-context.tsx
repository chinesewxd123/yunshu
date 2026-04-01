import { createContext, useContext, useEffect, useState } from "react";
import type { PropsWithChildren } from "react";
import type { EmailLoginPayload, LoginResult, PasswordLoginPayload, UserItem } from "../types/api";
import { emailLogin, getCurrentUser, logout as logoutRequest, passwordLogin } from "../services/auth";
import { clearAuthStorage, getToken, getUser, setToken, setUser } from "../services/storage";

interface AuthContextValue {
  user: UserItem | null;
  token: string;
  loading: boolean;
  isAuthenticated: boolean;
  passwordLoginAction: (payload: PasswordLoginPayload) => Promise<void>;
  emailLoginAction: (payload: EmailLoginPayload) => Promise<void>;
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

  function applyLoginResult(result: LoginResult) {
    setToken(result.token);
    setUser(result.user);
    setTokenState(result.token);
    setUserState(result.user);
  }

  async function passwordLoginAction(payload: PasswordLoginPayload) {
    const result = await passwordLogin(payload);
    applyLoginResult(result);
  }

  async function emailLoginAction(payload: EmailLoginPayload) {
    const result = await emailLogin(payload);
    applyLoginResult(result);
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
        passwordLoginAction,
        emailLoginAction,
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
