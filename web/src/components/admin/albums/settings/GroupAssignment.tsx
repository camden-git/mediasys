import { useEffect, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { queryKeys } from '../../../../lib/queryKeys';
import { useAlbumData } from '../../../../store/albumContextHooks';
import { invalidateAlbums } from '../../../../lib/queryClient';
import FlashMessageRender from '../../../elements/FlashMessageRender';
import { useUIStore } from '../../../../store/useUIStore';
import { Button } from '../../../elements/Button';
import HeaderedContent from '../../../elements/HeaderedContent.tsx';
import { Field, Label, Description } from '../../../elements/Fieldset';
import { Select } from '../../../elements/Select';
import { listGroups, setAlbumGroup } from '../../../../api/admin/groups';

export function GroupAssignment() {
    const album = useAlbumData();
    const albumId = album.id;
    const addFlash = useUIStore((s) => s.addFlash);

    const {
        data: groups = [],
        isLoading: isLoadingGroups,
        isError: groupsFailed,
        error: groupsError,
        refetch: refetchGroups,
    } = useQuery({ queryKey: queryKeys.groups.adminList(), queryFn: listGroups });
    const [selectedGroupId, setSelectedGroupId] = useState<string>(album.group_id?.toString() ?? '');
    const [isSaving, setIsSaving] = useState(false);

    // Keep select in sync if album context changes
    useEffect(() => {
        setSelectedGroupId(album.group_id?.toString() ?? '');
    }, [album.group_id]);

    const handleSave = async () => {
        setIsSaving(true);
        try {
            const groupId = selectedGroupId ? parseInt(selectedGroupId, 10) : null;
            await setAlbumGroup(albumId, groupId);
            await invalidateAlbums();
            addFlash({
                key: `album-${albumId}`,
                type: 'success',
                title: 'Group Updated',
                message: groupId ? 'Album assigned to group.' : 'Album removed from group.',
            });
        } catch (error: any) {
            addFlash({
                key: `album-${albumId}`,
                type: 'error',
                title: 'Error',
                message: error.message || 'Failed to update group assignment.',
            });
        } finally {
            setIsSaving(false);
        }
    };

    return (
        <HeaderedContent
            title={'Album Group'}
            description={'Assign this album to a group collection shown on the home page.'}
            className={'mt-16 pb-8'}
        >
            <FlashMessageRender byKey={`album-${albumId}`} />
            {groupsFailed && (
                <div role='alert' className='mb-4 flex items-center gap-3 text-sm text-red-600'>
                    <span>Could not load groups: {(groupsError as Error)?.message}</span>
                    <Button type='button' plain onClick={() => refetchGroups()}>
                        Retry
                    </Button>
                </div>
            )}
            <Field>
                <Label htmlFor='group_id'>Group</Label>
                <Select
                    id='group_id'
                    name='group_id'
                    value={selectedGroupId}
                    onChange={(e) => setSelectedGroupId(e.target.value)}
                    disabled={isLoadingGroups || isSaving}
                >
                    <option value=''>No group</option>
                    {isLoadingGroups && (
                        <option value='' disabled>
                            Loading groups…
                        </option>
                    )}
                    {groups.map((group) => (
                        <option key={group.id} value={group.id.toString()}>
                            {group.name}
                        </option>
                    ))}
                </Select>
                <Description>Select a group or leave blank to remove this album from any group.</Description>
            </Field>

            <div className='mt-4 flex justify-end'>
                <Button onClick={handleSave} disabled={isSaving || isLoadingGroups || groupsFailed}>
                    {isSaving ? 'Saving…' : 'Save Group'}
                </Button>
            </div>
        </HeaderedContent>
    );
}
