import { create } from 'zustand';
import * as api from '../api';
import { User, LoginPayload, RegisterPayload, AuthResponse } from '../types';
import { Role } from '../types';

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
    token: localStorage.getItem('authToken'),
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
        if (token) {
            localStorage.setItem('authToken', token);
        } else {
            localStorage.removeItem('authToken');
        }
        set({ token });
    },

    clearAuth: () => {
        localStorage.removeItem('authToken');
        set({ user: null, token: null });
    },

    setIsInitializing: (isInitializing) => set({ isInitializing }),

    login: async (payload) => {
        const response: AuthResponse = await api.loginUser(payload);
        get().setToken(response.token);
        set({ user: response.user as AuthenticatedUser });
    },

    register: async (payload) => {
        await api.registerUser(payload);
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
            const user: User = await api.getCurrentUser();
            set({ user: user as AuthenticatedUser });
        } catch {
            get().clearAuth();
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
