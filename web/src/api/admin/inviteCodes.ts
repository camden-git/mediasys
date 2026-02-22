import http from '../http';
import { AdminInviteCodeResponse, InviteCodeCreatePayload } from '../../types';
import { ApiResponse, PaginatedResult, PaginationRequest, toPaginatedResult, toPaginationQuery } from '../standard';

export const listInviteCodes = async (
    params?: PaginationRequest,
): Promise<PaginatedResult<AdminInviteCodeResponse>> => {
    const response = await http.get<ApiResponse<AdminInviteCodeResponse[]>>('/admin/invite-codes', {
        params: toPaginationQuery(params),
    });
    return toPaginatedResult(response.data);
};

export const createInviteCode = async (payload: InviteCodeCreatePayload): Promise<AdminInviteCodeResponse> => {
    const response = await http.post<ApiResponse<AdminInviteCodeResponse>>('/admin/invite-codes', payload);
    return response.data.data;
};

export const deleteInviteCode = async (codeId: number): Promise<void> => {
    await http.delete(`/admin/invite-codes/${codeId}`);
};
