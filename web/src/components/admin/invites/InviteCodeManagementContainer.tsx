import React, { useEffect } from 'react';
import { CreateInviteCodeForm } from './CreateInviteCodeForm.tsx';
import { Heading } from '../../elements/Heading.tsx';
import PageContentBlock from '../../elements/PageContentBlock.tsx';
import { Can } from '../../elements/Can.tsx';
import LoadingSpinner from '../../elements/LoadingSpinner.tsx';
import { useInviteCodes } from '../../../api/swr/useInviteCodes';
import { useFlash } from '../../../hooks/useFlash';
import FlashMessageRender from '../../elements/FlashMessageRender.tsx';
import { Text, TextLink } from '../../elements/Text.tsx';
import {Table, TableBody, TableHead, TableHeader, TableRow} from "../../elements/Table.tsx";
import InviteCodeRow from "./InviteCodeRow.tsx";

const InviteCodeManagementContainer: React.FC = () => {
    const { data: inviteCodes, error, isValidating } = useInviteCodes();
    const { clearFlashes, clearAndAddHttpError } = useFlash();

    useEffect(() => {
        if (!error) {
            clearFlashes('invite-codes');
            return;
        }

        clearAndAddHttpError({ error, key: 'invite-codes' });
    }, [error, clearFlashes, clearAndAddHttpError]);

    if (!inviteCodes || (error && isValidating)) {
        return <LoadingSpinner />;
    }

    return (
        <PageContentBlock title={'Invite Codes'}>
            <div className='mb-16 flex w-full flex-wrap items-end justify-between gap-4'>
                <div>
                    <Heading>Invite Codes</Heading>
                    <Text className={'mt-1'}>
                        Invite Codes allow users to <TextLink to={'/auth/register'}>register</TextLink> without needing
                        to have their account manually created by an admin.
                    </Text>
                </div>
                <div className='flex gap-4'>
                    <Can permission={'invite.create'}>
                        <CreateInviteCodeForm />
                    </Can>
                </div>
            </div>
            <FlashMessageRender byKey={'invite-codes'} className={'mb-4'} />
            <Can permission={'invite.list'}>
                <Table>
                    <TableHead>
                        <TableRow>
                            <TableHeader>Code</TableHeader>
                            <TableHeader>Uses</TableHeader>
                            <TableHeader>Expires At</TableHeader>
                            <TableHeader>Created At</TableHeader>
                        </TableRow>
                    </TableHead>
                    <TableBody>
                        {inviteCodes.map((inviteCode) => (
                            <InviteCodeRow key={inviteCode.id} inviteCode={inviteCode} />
                        ))}
                    </TableBody>
                </Table>
            </Can>
        </PageContentBlock>
    );
};

export default InviteCodeManagementContainer;
