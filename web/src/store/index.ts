import { createStore, action, thunk, Action, Thunk } from 'easy-peasy';
import { Album, DirectoryListing } from '../types';
import { getAlbums, getAlbumDetails, getAlbumContents } from '../api';
import authModel, { AuthModel } from './authModel';
import uiStore, { UIStore } from './uiStore';
import inviteCodeStore, { InviteCodeStore } from './inviteCodeStore';
import roleStore, { RoleStore } from './roleStore';
import userStore, { UserStore } from './userStore';
import progressStore, { ProgressStore } from './progressStore';
import adminAlbumStore, { AdminAlbumStore } from './adminAlbumStore';
import albumContextStore, { AlbumContextStore } from './albumContext';

export interface AlbumListModel {
    items: Album[];
    setItems: Action<AlbumListModel, Album[]>;
    isLoading: boolean;
    setIsLoading: Action<AlbumListModel, boolean>;
    error: string | null;
    setError: Action<AlbumListModel, string | null>;
    fetchAlbums: Thunk<AlbumListModel>;
}

const albumListModel: AlbumListModel = {
    items: [],
    isLoading: false,
    error: null,
    setItems: action((state, payload) => {
        state.items = payload;
    }),
    setIsLoading: action((state, payload) => {
        state.isLoading = payload;
    }),
    setError: action((state, payload) => {
        state.error = payload;
    }),
    fetchAlbums: thunk(async (actions) => {
        actions.setIsLoading(true);
        actions.setError(null);
        try {
            const albums = await getAlbums();
            actions.setItems(albums);
        } catch (error: any) {
            console.error('Failed to fetch albums:', error);
            actions.setError(error.message || 'Failed to fetch albums');
        } finally {
            actions.setIsLoading(false);
        }
    }),
};

export interface ContentViewModel {
    currentIdentifier: string | null;
    currentAlbum: Album | null;
    directoryListing: DirectoryListing | null;
    setCurrentAlbum: Action<ContentViewModel, Album | null>;
    setDirectoryListing: Action<ContentViewModel, DirectoryListing | null>;
    clearViewData: Action<ContentViewModel>;

    isLoading: boolean;
    setIsLoading: Action<ContentViewModel, boolean>;
    error: string | null;
    setError: Action<ContentViewModel, string | null>;

    fetchAlbumDataAndContents: Thunk<ContentViewModel, string>;
}

const contentViewModel: ContentViewModel = {
    currentIdentifier: null,
    currentAlbum: null,
    directoryListing: null,
    isLoading: false,
    error: null,

    setCurrentAlbum: action((state, payload) => {
        state.currentAlbum = payload;
        if (payload) {
            state.currentIdentifier = payload.slug ?? payload.id.toString();
        }
    }),
    setDirectoryListing: action((state, payload) => {
        state.directoryListing = payload;
    }),
    clearViewData: action((state) => {
        state.currentIdentifier = null;
        state.currentAlbum = null;
        state.directoryListing = null;
        state.isLoading = false;
        state.error = null;
    }),
    setIsLoading: action((state, payload) => {
        state.isLoading = payload;
    }),
    setError: action((state, payload) => {
        state.error = payload;
    }),

    fetchAlbumDataAndContents: thunk(async (actions, identifier) => {
        actions.setIsLoading(true);
        actions.setError(null);
        actions.clearViewData();
        try {
            const [albumDetails, albumContents] = await Promise.all([
                getAlbumDetails(identifier),
                getAlbumContents(identifier),
            ]);

            actions.setCurrentAlbum(albumDetails);
            actions.setDirectoryListing(albumContents);
        } catch (error: any) {
            console.error(`Failed to fetch data for album ${identifier}:`, error);
            actions.setError(error.message || `Failed to fetch data for album ${identifier}`);
            actions.clearViewData();
        } finally {
            actions.setIsLoading(false);
        }
    }),
};

export interface StoreModel {
    albumList: AlbumListModel;
    contentView: ContentViewModel;
    auth: AuthModel;
    ui: UIStore;
    inviteCodes: InviteCodeStore;
    roles: RoleStore;
    users: UserStore;
    progress: ProgressStore;
    adminAlbums: AdminAlbumStore;
    albumContext: AlbumContextStore;
}

const rootStoreModel: StoreModel = {
    albumList: albumListModel,
    contentView: contentViewModel,
    auth: authModel,
    ui: uiStore,
    inviteCodes: inviteCodeStore,
    roles: roleStore,
    users: userStore,
    progress: progressStore,
    adminAlbums: adminAlbumStore,
    albumContext: albumContextStore,
};

const store = createStore(rootStoreModel);

export default store;
