import { UpdateAlbumForm } from './UpdateAlbumForm';
import { BannerUpload } from './BannerUpload';
import { DeleteAlbumSection } from './DeleteAlbumSection';
import { GroupAssignment } from './GroupAssignment';
import { DefaultTagsSection } from './DefaultTagsSection';
import { Can } from '../../../elements/Can';
import { useAlbumData } from '../../../../store/albumContextHooks';

// Mirrors the backend guards: album update/delete and group assignment are global-only,
// banners and default tags also accept album-scoped photo metadata editing.
const META_PERMISSIONS = ['album.edit.general', 'album.photo.editmeta'];

export function SettingsContainer() {
    const album = useAlbumData();

    return (
        <div className='mx-auto max-w-7xl divide-y divide-zinc-950/10'>
            <Can permission='album.edit.general'>
                <UpdateAlbumForm />
            </Can>
            <Can permission={META_PERMISSIONS} albumId={album.id}>
                <BannerUpload />
            </Can>
            <Can permission='album.group.manage'>
                <GroupAssignment />
            </Can>
            <Can permission={META_PERMISSIONS} albumId={album.id}>
                <DefaultTagsSection />
            </Can>
            <Can permission='album.delete'>
                <DeleteAlbumSection />
            </Can>
        </div>
    );
}
