import { UpdateAlbumForm } from './UpdateAlbumForm';
import { BannerUpload } from './BannerUpload';
import { DeleteAlbumSection } from './DeleteAlbumSection';
import { GroupAssignment } from './GroupAssignment';
import { DefaultTagsSection } from './DefaultTagsSection';

export function SettingsContainer() {
    return (
        <div className='mx-auto max-w-7xl divide-y divide-zinc-950/10'>
            <UpdateAlbumForm />
            <BannerUpload />
            <GroupAssignment />
            <DefaultTagsSection />
            <DeleteAlbumSection />
        </div>
    );
}
