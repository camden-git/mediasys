import { Action, action } from 'easy-peasy';
import { AdminRoleResponse } from '../types';

export interface RoleStore {
    data: AdminRoleResponse[];
    setRoles: Action<RoleStore, AdminRoleResponse[]>;
    appendRole: Action<RoleStore, AdminRoleResponse>;
    updateRole: Action<RoleStore, AdminRoleResponse>;
    removeRole: Action<RoleStore, number>;
}

const roleStore: RoleStore = {
    data: [],

    setRoles: action((state, payload) => {
        state.data = payload;
    }),

    appendRole: action((state, payload) => {
        if (state.data.find((role) => role.id === payload.id)) {
            state.data = state.data.map((role) => (role.id === payload.id ? payload : role));
        } else {
            state.data = [...state.data, payload];
        }
    }),

    updateRole: action((state, payload) => {
        const index = state.data.findIndex((role) => role.id === payload.id);
        if (index !== -1) {
            state.data[index] = payload;
        }
    }),

    removeRole: action((state, payload) => {
        state.data = state.data.filter((role) => role.id !== payload);
    }),
};

export default roleStore;
