import { UpdateAlbumForm } from './UpdateAlbumForm';
import { BannerUpload } from './BannerUpload';

export function SettingsContainer() {
    return (
        <div className='mx-auto max-w-7xl divide-y divide-zinc-950/10'>
            <UpdateAlbumForm />
            <BannerUpload />
        </div>
    );
}
