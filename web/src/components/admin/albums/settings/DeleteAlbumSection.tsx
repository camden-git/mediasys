import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAlbumData } from '../../../../store/albumContextHooks';
import { invalidateAlbums } from '../../../../lib/queryClient';
import FlashMessageRender from '../../../elements/FlashMessageRender';
import { useUIStore } from '../../../../store/useUIStore';
import { deleteAlbum as deleteAlbumAPI } from '../../../../api/admin/albums';
import { Button } from '../../../elements/Button';
import { Dialog, DialogActions, DialogBody, DialogDescription, DialogTitle } from '../../../elements/Dialog';
import { Field, Label } from '../../../elements/Fieldset';
import { Input } from '../../../elements/Input';
import HeaderedContent from '../../../elements/HeaderedContent';

export function DeleteAlbumSection() {
    const album = useAlbumData();
    const addFlash = useUIStore((s) => s.addFlash);
    const navigate = useNavigate();

    const [isOpen, setIsOpen] = useState(false);
    const [confirmName, setConfirmName] = useState('');

    const handleDelete = async () => {
        try {
            await deleteAlbumAPI(album.id);
            // Mark stale without refetching the album that no longer exists; the list refetches on mount.
            await invalidateAlbums('none');
            addFlash({
                key: 'album-deleted',
                type: 'success',
                title: 'Deleted',
                message: 'Album deleted successfully',
            });
            setIsOpen(false);
            navigate('/admin/albums');
        } catch (err: any) {
            addFlash({
                key: 'album-deleted-error',
                type: 'error',
                title: 'Error',
                message: err.message || 'Failed to delete album',
            });
        }
    };

    const handleClose = () => {
        setIsOpen(false);
        setConfirmName('');
    };

    return (
        <HeaderedContent
            title='Danger Zone'
            description='Permanently delete this album and all associated data. This action cannot be undone.'
            className={'mt-16 pb-8'}
        >
            <FlashMessageRender byKey='album-deleted-error' />
            <Button color='red' onClick={() => setIsOpen(true)}>
                Delete Album
            </Button>
            <Dialog open={isOpen} onClose={handleClose}>
                <DialogTitle>Delete Album</DialogTitle>
                <DialogDescription>
                    This will permanently delete <strong>{album.name}</strong> and all its data. To confirm, type the
                    full album name below.
                </DialogDescription>
                <DialogBody>
                    <Field>
                        <Label>Album name</Label>
                        <Input
                            type='text'
                            value={confirmName}
                            onChange={(e) => setConfirmName(e.target.value)}
                            placeholder={album.name}
                            autoComplete='off'
                        />
                    </Field>
                </DialogBody>
                <DialogActions>
                    <Button plain onClick={handleClose}>
                        Cancel
                    </Button>
                    <Button color='red' disabled={confirmName !== album.name} onClick={handleDelete}>
                        Delete
                    </Button>
                </DialogActions>
            </Dialog>
        </HeaderedContent>
    );
}
