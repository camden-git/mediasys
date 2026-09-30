import { create } from 'zustand';
import { loginUser, registerUser, getCurrentUser } from '../api/auth';
import { getAuthToken, setAuthToken } from '../api/token';
import { errorStatus } from '../api/errors';
import { User, LoginPayload, RegisterPayload, AuthResponse } from '../types';
import { Role } from '../types';
import { queryClient } from '../lib/queryClient';

export interface AuthenticatedUser extends User {
    roles: Role[];
    global_permissions: string[];
}

interface AuthState {
    user: AuthenticatedUser | null;
    token: string | null;
    isInitializing: boolean;
    isAuthenticated: () => boolean;
    currentUserPermissions: () => string[];
    setUser: (user: AuthenticatedUser | null) => void;
    setToken: (token: string | null) => void;
    clearAuth: () => void;
    setIsInitializing: (v: boolean) => void;
    login: (payload: LoginPayload) => Promise<void>;
    register: (payload: RegisterPayload) => Promise<void>;
    logout: () => void;
    fetchCurrentUser: () => Promise<void>;
    initializeAuth: () => Promise<void>;
}

export const useAuthStore = create<AuthState>()((set, get) => ({
    user: null,
    token: getAuthToken(),
    isInitializing: true,

    isAuthenticated: () => !!(get().user && get().token),

    currentUserPermissions: () => {
        const { user } = get();
        if (!user) return [];
        const permissions = new Set<string>();
        user.global_permissions?.forEach((p) => permissions.add(p));
        user.roles?.forEach((role) => {
            role.global_permissions?.forEach((p) => permissions.add(p));
        });
        return Array.from(permissions);
    },

    setUser: (user) => set({ user }),

    setToken: (token) => {
        setAuthToken(token);
        set({ token });
    },

    clearAuth: () => {
        setAuthToken(null);
        // Drop cached server data so the next user never sees the previous user's data.
        queryClient.clear();
        set({ user: null, token: null });
    },

    setIsInitializing: (isInitializing) => set({ isInitializing }),

    login: async (payload) => {
        const response: AuthResponse = await loginUser(payload);
        queryClient.clear();
        get().setToken(response.token);
        set({ user: response.user as AuthenticatedUser });
    },

    register: async (payload) => {
        await registerUser(payload);
    },

    logout: () => {
        get().clearAuth();
    },

    fetchCurrentUser: async () => {
        if (!get().token) {
            get().clearAuth();
            return;
        }
        try {
            const user: User = await getCurrentUser();
            set({ user: user as AuthenticatedUser });
        } catch (error: unknown) {
            // Only an invalid/expired token signs the user out; network blips and 5xx keep the session.
            if (errorStatus(error) === 401) {
                get().clearAuth();
            }
        }
    },

    initializeAuth: async () => {
        set({ isInitializing: true });
        try {
            if (get().token) {
                await get().fetchCurrentUser();
            } else {
                get().clearAuth();
            }
        } catch {
            get().clearAuth();
        } finally {
            set({ isInitializing: false });
        }
    },
}));

/** Re-fetch the signed-in user's roles/permissions after a change that may have affected them. */
export const refreshAuthUser = (userId?: number) => {
    const { user, fetchCurrentUser } = useAuthStore.getState();
    if (user && (userId === undefined || user.id === userId)) {
        void fetchCurrentUser();
    }
};
