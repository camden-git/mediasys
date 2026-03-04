import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useStoreActions, useStoreState } from '../../../../store/hooks';
import { Button } from '../../../elements/Button';
import { Dialog, DialogActions, DialogBody, DialogDescription, DialogTitle } from '../../../elements/Dialog';
import { Field, Label } from '../../../elements/Fieldset';
import { Input } from '../../../elements/Input';
import HeaderedContent from '../../../elements/HeaderedContent';

export function DeleteAlbumSection() {
    const album = useStoreState((state) => state.albumContext.data!);
    const { deleteAlbum } = useStoreActions((actions) => actions.adminAlbums);
    const { addFlash } = useStoreActions((actions) => actions.ui);
    const navigate = useNavigate();

    const [isOpen, setIsOpen] = useState(false);
    const [confirmName, setConfirmName] = useState('');

    const handleDelete = () => {
        deleteAlbum({
            id: album.id,
            addFlash,
            onSuccess: () => {
                setIsOpen(false);
                navigate('/admin/albums');
            },
        });
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
