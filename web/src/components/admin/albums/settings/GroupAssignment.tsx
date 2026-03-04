import { useEffect, useState } from 'react';
import { useAlbumContextStore } from '../../../../store/useAlbumContextStore';
import { useUIStore } from '../../../../store/useUIStore';
import { Button } from '../../../elements/Button';
import HeaderedContent from '../../../elements/HeaderedContent.tsx';
import { Field, Label, Description } from '../../../elements/Fieldset';
import { Select } from '../../../elements/Select';
import { listGroups, setAlbumGroup } from '../../../../api/admin/groups';
import { AlbumGroup } from '../../../../types';

export function GroupAssignment() {
    const album = useAlbumContextStore((s) => s.data!);
    const albumId = album.id;
    const addFlash = useUIStore((s) => s.addFlash);
    const setAlbum = useAlbumContextStore((s) => s.setAlbum);

    const [groups, setGroups] = useState<AlbumGroup[]>([]);
    const [selectedGroupId, setSelectedGroupId] = useState<string>(album.group_id?.toString() ?? '');
    const [isSaving, setIsSaving] = useState(false);
    const [isLoadingGroups, setIsLoadingGroups] = useState(true);

    useEffect(() => {
        (async () => {
            try {
                const data = await listGroups();
                setGroups(data);
            } catch {
                // silently fail — group list is non-critical
            } finally {
                setIsLoadingGroups(false);
            }
        })();
    }, []);

    // Keep select in sync if album context changes
    useEffect(() => {
        setSelectedGroupId(album.group_id?.toString() ?? '');
    }, [album.group_id]);

    const handleSave = async () => {
        setIsSaving(true);
        try {
            const groupId = selectedGroupId ? parseInt(selectedGroupId, 10) : null;
            await setAlbumGroup(albumId, groupId);
            setAlbum({ ...album, group_id: groupId ?? undefined });
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
                <Button onClick={handleSave} disabled={isSaving || isLoadingGroups}>
                    {isSaving ? 'Saving…' : 'Save Group'}
                </Button>
            </div>
        </HeaderedContent>
    );
}
