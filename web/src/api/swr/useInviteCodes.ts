import useSWR from 'swr';
import { listInviteCodes } from '../admin/inviteCodes';
import { AdminInviteCodeResponse } from '../../types';

export const useInviteCodes = () => {
    return useSWR<AdminInviteCodeResponse[]>('invite-codes', listInviteCodes);
};
