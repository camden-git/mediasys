import { Action, action } from 'easy-peasy';
import { AdminUserResponse } from '../types';

export interface UserStore {
    data: AdminUserResponse[];
    setUsers: Action<UserStore, AdminUserResponse[]>;
    appendUser: Action<UserStore, AdminUserResponse>;
    updateUser: Action<UserStore, AdminUserResponse>;
    removeUser: Action<UserStore, number>;
}

const userStore: UserStore = {
    data: [],

    setUsers: action((state, payload) => {
        state.data = payload;
    }),

    appendUser: action((state, payload) => {
        if (state.data.find((user) => user.id === payload.id)) {
            state.data = state.data.map((user) => (user.id === payload.id ? payload : user));
        } else {
            state.data = [...state.data, payload];
        }
    }),

    updateUser: action((state, payload) => {
        const index = state.data.findIndex((user) => user.id === payload.id);
        if (index !== -1) {
            state.data[index] = payload;
        }
    }),

    removeUser: action((state, payload) => {
        state.data = state.data.filter((user) => user.id !== payload);
    }),
};

export default userStore;
