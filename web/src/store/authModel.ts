import { Action, Thunk, action, thunk, computed, Computed } from 'easy-peasy';
import * as api from '../api';
import { User, LoginPayload, RegisterPayload, AuthResponse } from '../types';
import { Role } from '../types';

export interface AuthenticatedUser extends User {
    roles: Role[];
    global_permissions: string[];
    // album_permissions are fetched on demand
}

export interface AuthModel {
    // state
    user: AuthenticatedUser | null;
    token: string | null;
    isInitializing: boolean;
    isAuthenticated: Computed<AuthModel, boolean>;

    // actions
    setUser: Action<AuthModel, AuthenticatedUser | null>;
    setToken: Action<AuthModel, string | null>;
    clearAuth: Action<AuthModel>;
    setIsInitializing: Action<AuthModel, boolean>;

    // thunks
    login: Thunk<AuthModel, LoginPayload, void, Promise<void>>;
    register: Thunk<AuthModel, RegisterPayload, void, Promise<void>>;
    logout: Thunk<AuthModel>;
    fetchCurrentUser: Thunk<AuthModel, void, any, Promise<void>>;
    initializeAuth: Thunk<AuthModel, void, any, Promise<void>>;

    // computed for permissions
    currentUserPermissions: Computed<AuthModel, string[]>;
}

const authModel: AuthModel = {
    // state
    user: null,
    token: localStorage.getItem('authToken'),
    isInitializing: true,

    isAuthenticated: computed((state) => !!state.user && !!state.token),

    // actions
    setUser: action((state, payload) => {
        state.user = payload;
    }),
    setToken: action((state, token) => {
        state.token = token;
        if (token) {
            localStorage.setItem('authToken', token);
        } else {
            localStorage.removeItem('authToken');
        }
    }),
    clearAuth: action((state) => {
        state.user = null;
        state.token = null;
        localStorage.removeItem('authToken');
    }),
    setIsInitializing: action((state, payload) => {
        state.isInitializing = payload;
    }),

    // thunks
    login: thunk(async (actions, payload) => {
        try {
            const response: AuthResponse = await api.loginUser(payload);
            actions.setToken(response.token);
            actions.setUser(response.user as AuthenticatedUser);
        } catch (err: any) {
            console.error('Login failed:', err);
            actions.clearAuth();
            throw err;
        }
    }),
    register: thunk(async (_actions, payload) => {
        try {
            await api.registerUser(payload);
        } catch (err: any) {
            console.error('Registration failed:', err);
            throw err;
        }
    }),
    logout: thunk((actions) => {
        actions.clearAuth();
    }),
    fetchCurrentUser: thunk(async (actions, _, { getState }) => {
        if (!getState().token) {
            actions.clearAuth();
            return;
        }
        try {
            const user: User = await api.getCurrentUser();
            actions.setUser(user as AuthenticatedUser);
        } catch (err: any) {
            console.error('Failed to fetch current user:', err);
            actions.clearAuth(); // the token might be invalid or expired
        }
    }),
    initializeAuth: thunk(async (actions, _, { getState }) => {
        actions.setIsInitializing(true);
        try {
            if (getState().token) {
                await actions.fetchCurrentUser();
            } else {
                actions.clearAuth();
            }
        } catch (error) {
            console.error('Error during auth initialization:', error);
            actions.clearAuth();
        } finally {
            actions.setIsInitializing(false);
        }
    }),

    // computed for permissions
    currentUserPermissions: computed((state) => {
        if (!state.user) {
            return [];
        }
        const permissions = new Set<string>();
        state.user.global_permissions?.forEach((p) => {
            permissions.add(p);
        });

        state.user.roles?.forEach((role) => {
            role.global_permissions?.forEach((p) => {
                permissions.add(p);
            });
        });
        return Array.from(permissions);
    }),
};

export default authModel;
