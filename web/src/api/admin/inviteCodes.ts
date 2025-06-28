import http from '../http';
import { InviteCodeResponse, InviteCodeCreatePayload } from '../../types';

export const listInviteCodes = async (): Promise<InviteCodeResponse[]> => {
    const response = await http.get('/admin/invite-codes');
    return response.data;
};

export const createInviteCode = async (payload: InviteCodeCreatePayload): Promise<InviteCodeResponse> => {
    const response = await http.post('/admin/invite-codes', payload);
    return response.data;
};

export const deleteInviteCode = async (codeId: number): Promise<void> => {
    await http.delete(`/admin/invite-codes/${codeId}`);
};
