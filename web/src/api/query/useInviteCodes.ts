import { useQuery } from '@tanstack/react-query';
import { listInviteCodes } from '../admin/inviteCodes';
import { AdminInviteCodeResponse } from '../../types';
import { PaginatedResult, PaginationRequest } from '../standard';
import { queryKeys } from '../../lib/queryKeys';

export const useInviteCodes = (params?: PaginationRequest) => {
    return useQuery<PaginatedResult<AdminInviteCodeResponse>>({
        queryKey: queryKeys.inviteCodes.list(params),
        queryFn: ({ signal }) => listInviteCodes(params, signal),
    });
};
