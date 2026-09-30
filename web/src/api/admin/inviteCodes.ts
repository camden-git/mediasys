import http from '../http';
import { AdminInviteCodeResponse, InviteCodeCreatePayload } from '../../types';
import { ApiResponse, PaginatedResult, PaginationRequest, toPaginatedResult, toPaginationQuery } from '../standard';

export const listInviteCodes = async (
    params?: PaginationRequest,
    signal?: AbortSignal,
): Promise<PaginatedResult<AdminInviteCodeResponse>> => {
    const response = await http.get<ApiResponse<AdminInviteCodeResponse[]>>('/admin/invite-codes', {
        params: toPaginationQuery(params),
        signal,
    });
    return toPaginatedResult(response.data);
};

export const createInviteCode = async (payload: InviteCodeCreatePayload): Promise<AdminInviteCodeResponse> => {
    const response = await http.post<ApiResponse<AdminInviteCodeResponse>>('/admin/invite-codes', payload);
    return response.data.data;
};
