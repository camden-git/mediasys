import { Action, action } from 'easy-peasy';
import { AdminInviteCodeResponse } from '../types';

export interface InviteCodeStore {
    data: AdminInviteCodeResponse[];
    setInviteCodes: Action<InviteCodeStore, AdminInviteCodeResponse[]>;
    appendInviteCode: Action<InviteCodeStore, AdminInviteCodeResponse>;
    removeInviteCode: Action<InviteCodeStore, number>;
}

const inviteCodeStore: InviteCodeStore = {
    data: [],

    setInviteCodes: action((state, payload) => {
        state.data = payload;
    }),

    appendInviteCode: action((state, payload) => {
        if (state.data.find((inviteCode) => inviteCode.id === payload.id)) {
            state.data = state.data.map((inviteCode) => (inviteCode.id === payload.id ? payload : inviteCode));
        } else {
            state.data = [...state.data, payload];
        }
    }),

    removeInviteCode: action((state, payload) => {
        state.data = state.data.filter((inviteCode) => inviteCode.id !== payload);
    }),
};

export default inviteCodeStore;
