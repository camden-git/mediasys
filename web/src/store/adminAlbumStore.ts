import { Action, action } from 'easy-peasy';
import { CreateAlbumPayload, UpdateAlbumPayload, uploadAlbumBanner } from '../api/admin/albums';
import { createAlbum, deleteAlbum, updateAlbum } from '../api/admin/albums';

export interface AdminAlbumStore {
    // actions
    createAlbum: Action<AdminAlbumStore, { payload: CreateAlbumPayload; onSuccess?: () => void; addFlash: any }>;
    updateAlbum: Action<
        AdminAlbumStore,
        { id: number; payload: UpdateAlbumPayload; onSuccess?: () => void; addFlash: any; setAlbum?: any }
    >;
    deleteAlbum: Action<AdminAlbumStore, { id: number; onSuccess?: () => void; addFlash: any }>;
    uploadAlbumBanner: Action<
        AdminAlbumStore,
        { id: number; file: File; onSuccess?: () => void; addFlash: any; setAlbum?: any }
    >;
}

const adminAlbumStore: AdminAlbumStore = {
    createAlbum: action((state, { payload, onSuccess, addFlash }) => {
        createAlbum(payload)
            .then(() => {
                addFlash({
                    key: 'album-created',
                    type: 'success',
                    message: 'Album created successfully',
                });
                onSuccess?.();
            })
            .catch((error) => {
                addFlash({
                    key: 'album-created-error',
                    type: 'error',
                    message: error.response?.data?.error || 'Failed to create album',
                });
            });
    }),

    updateAlbum: action((state, { id, payload, onSuccess, addFlash, setAlbum }) => {
        updateAlbum(id, payload)
            .then((updatedAlbum) => {
                addFlash({
                    key: 'album-updated',
                    type: 'success',
                    message: 'Album updated successfully',
                });

                if (setAlbum) {
                    setAlbum(updatedAlbum);
                }
                onSuccess?.();
            })
            .catch((error) => {
                addFlash({
                    key: 'album-updated-error',
                    type: 'error',
                    message: error.response?.data?.error || 'Failed to update album',
                });
            });
    }),

    deleteAlbum: action((state, { id, onSuccess, addFlash }) => {
        deleteAlbum(id)
            .then(() => {
                addFlash({
                    key: 'album-deleted',
                    type: 'success',
                    message: 'Album deleted successfully',
                });
                onSuccess?.();
            })
            .catch((error) => {
                addFlash({
                    key: 'album-deleted-error',
                    type: 'error',
                    message: error.response?.data?.error || 'Failed to delete album',
                });
            });
    }),

    uploadAlbumBanner: action((state, { id, file, onSuccess, addFlash, setAlbum }) => {
        uploadAlbumBanner(id, file)
            .then((updatedAlbum) => {
                addFlash({
                    key: 'album-banner-uploaded',
                    type: 'success',
                    message: 'Album banner uploaded successfully',
                });

                if (setAlbum) {
                    setAlbum(updatedAlbum);
                }
                onSuccess?.();
            })
            .catch((error) => {
                addFlash({
                    key: 'album-banner-upload-error',
                    type: 'error',
                    message: error.response?.data?.error || 'Failed to upload album banner',
                });
            });
    }),
};

export default adminAlbumStore;
