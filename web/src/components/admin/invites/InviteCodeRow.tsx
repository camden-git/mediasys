import React from 'react';
import { TableCell, TableRow } from '../../elements/Table.tsx';
import { AdminInviteCodeResponse } from '../../../types.ts';
import DateDisplay from '../../elements/DateDisplay.tsx';

interface InviteCodeRowProps {
    inviteCode: AdminInviteCodeResponse;
}

const InviteCodeRow: React.FC<InviteCodeRowProps> = ({ inviteCode }) => {
    return (
        <TableRow key={inviteCode.id}>
            <TableCell className='font-medium'>{inviteCode.code}</TableCell>
            <TableCell>
                {inviteCode.uses} / {inviteCode.max_uses ? inviteCode.max_uses : 'Unlimited'}
            </TableCell>
            <TableCell>
                {inviteCode.expires_at ? (
                    <DateDisplay dateString={inviteCode.expires_at} showCountdown={true} />
                ) : (
                    <span className='text-zinc-500'>Never expires</span>
                )}
            </TableCell>
            <TableCell>
                <DateDisplay dateString={inviteCode.created_at} showCountdown={false} />
            </TableCell>
        </TableRow>
    );
};

export default InviteCodeRow;



