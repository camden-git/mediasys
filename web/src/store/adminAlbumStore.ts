import { Action, action } from 'easy-peasy';
import {
    CreateAlbumPayload,
    UpdateAlbumPayload,
    addAlbumBanner,
    deleteAlbumBanner,
    reorderAlbumBanners,
} from '../api/admin/albums';
import { createAlbum, deleteAlbum, updateAlbum, getAlbum } from '../api/admin/albums';

export interface AdminAlbumStore {
    createAlbum: Action<AdminAlbumStore, { payload: CreateAlbumPayload; onSuccess?: () => void; addFlash: any }>;
    updateAlbum: Action<
        AdminAlbumStore,
        { id: number; payload: UpdateAlbumPayload; onSuccess?: () => void; addFlash: any; setAlbum?: any }
    >;
    deleteAlbum: Action<AdminAlbumStore, { id: number; onSuccess?: () => void; addFlash: any }>;
    addAlbumBanner: Action<
        AdminAlbumStore,
        { id: number; file: File; onSuccess?: () => void; addFlash: any; setAlbum?: any }
    >;
    deleteAlbumBanner: Action<
        AdminAlbumStore,
        { id: number; bannerId: number; onSuccess?: () => void; addFlash: any; setAlbum?: any }
    >;
    reorderAlbumBanners: Action<
        AdminAlbumStore,
        { id: number; bannerIds: number[]; onSuccess?: () => void; addFlash: any; setAlbum?: any }
    >;
}

const adminAlbumStore: AdminAlbumStore = {
    createAlbum: action((_state, { payload, onSuccess, addFlash }) => {
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

    updateAlbum: action((_state, { id, payload, onSuccess, addFlash, setAlbum }) => {
        updateAlbum(id, payload)
            .then((updatedAlbum) => {
                addFlash({
                    key: 'album-update',
                    type: 'success',
                    message: 'Album updated successfully',
                });
                if (setAlbum) setAlbum(updatedAlbum);
                onSuccess?.();
            })
            .catch((error) => {
                addFlash({
                    key: 'album-update',
                    type: 'error',
                    message: error.response?.data?.error || 'Failed to update album',
                });
            });
    }),

    deleteAlbum: action((_state, { id, onSuccess, addFlash }) => {
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

    addAlbumBanner: action((_state, { id, file, onSuccess, addFlash, setAlbum }) => {
        addAlbumBanner(id, file)
            .then(() => {
                addFlash({
                    key: 'album-banner-added',
                    type: 'success',
                    message: 'Banner added successfully',
                });
                // Refresh album to get updated banners list
                if (setAlbum) {
                    getAlbum(id)
                        .then(setAlbum)
                        .catch(() => {});
                }
                onSuccess?.();
            })
            .catch((error) => {
                addFlash({
                    key: 'album-banner-add-error',
                    type: 'error',
                    message: error.response?.data?.error || 'Failed to add banner',
                });
            });
    }),

    deleteAlbumBanner: action((_state, { id, bannerId, onSuccess, addFlash, setAlbum }) => {
        deleteAlbumBanner(id, bannerId)
            .then(() => {
                addFlash({
                    key: 'album-banner-deleted',
                    type: 'success',
                    message: 'Banner removed',
                });
                if (setAlbum) {
                    getAlbum(id)
                        .then(setAlbum)
                        .catch(() => {});
                }
                onSuccess?.();
            })
            .catch((error) => {
                addFlash({
                    key: 'album-banner-delete-error',
                    type: 'error',
                    message: error.response?.data?.error || 'Failed to remove banner',
                });
            });
    }),

    reorderAlbumBanners: action((_state, { id, bannerIds, onSuccess, addFlash, setAlbum }) => {
        reorderAlbumBanners(id, bannerIds)
            .then(() => {
                if (setAlbum) {
                    getAlbum(id)
                        .then(setAlbum)
                        .catch(() => {});
                }
                onSuccess?.();
            })
            .catch((error) => {
                addFlash({
                    key: 'album-banner-reorder-error',
                    type: 'error',
                    message: error.response?.data?.error || 'Failed to reorder banners',
                });
            });
    }),
};

export default adminAlbumStore;
