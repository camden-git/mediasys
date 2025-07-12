import http from '../http';
import { AdminInviteCodeResponse, InviteCodeCreatePayload } from '../../types';

export const listInviteCodes = async (): Promise<AdminInviteCodeResponse[]> => {
    const response = await http.get('/admin/invite-codes');
    return response.data;
};

export const createInviteCode = async (payload: InviteCodeCreatePayload): Promise<AdminInviteCodeResponse> => {
    const response = await http.post('/admin/invite-codes', payload);
    return response.data;
};

export const deleteInviteCode = async (codeId: number): Promise<void> => {
    await http.delete(`/admin/invite-codes/${codeId}`);
};
