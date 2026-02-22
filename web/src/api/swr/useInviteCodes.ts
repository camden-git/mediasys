import useSWR from 'swr';
import { listInviteCodes } from '../admin/inviteCodes';
import { AdminInviteCodeResponse } from '../../types';
import { PaginatedResult, PaginationRequest } from '../standard';

const inviteCodesKey = (params?: PaginationRequest) => [
    'invite-codes',
    params?.page ?? 1,
    params?.perPage ?? undefined,
];

export const useInviteCodes = (params?: PaginationRequest) => {
    return useSWR<PaginatedResult<AdminInviteCodeResponse>>(inviteCodesKey(params), () => listInviteCodes(params));
};
