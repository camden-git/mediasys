import { useAlbumData } from '../../../../store/albumContextHooks';
import { invalidateAlbums } from '../../../../lib/queryClient';
import FlashMessageRender from '../../../elements/FlashMessageRender';
import { useUIStore } from '../../../../store/useUIStore';
import HeaderedContent from '../../../elements/HeaderedContent.tsx';
import { addAlbumBanner, deleteAlbumBanner, reorderAlbumBanners } from '../../../../api/admin/albums';
import { BannerManager } from '../../shared/BannerManager';

export function BannerUpload() {
    const album = useAlbumData();
    const albumId = album.id;
    const banners = album.banners ?? [];
    const addFlash = useUIStore((s) => s.addFlash);

    const refreshAlbum = async () => {
        await invalidateAlbums();
    };

    const handleAdd = async (file: File) => {
        try {
            await addAlbumBanner(albumId, file);
            await refreshAlbum();
        } catch (error: any) {
            addFlash({
                key: 'banner-error',
                type: 'error',
                title: 'Error',
                message: error.message || 'Failed to add banner',
            });
        }
    };

    const handleDelete = async (bannerId: number) => {
        try {
            await deleteAlbumBanner(albumId, bannerId);
            await refreshAlbum();
        } catch (error: any) {
            addFlash({
                key: 'banner-error',
                type: 'error',
                title: 'Error',
                message: error.message || 'Failed to remove banner',
            });
        }
    };

    const handleReorder = async (bannerIds: number[]) => {
        try {
            await reorderAlbumBanners(albumId, bannerIds);
            await refreshAlbum();
        } catch (error: any) {
            addFlash({
                key: 'banner-error',
                type: 'error',
                title: 'Error',
                message: error.message || 'Failed to reorder banners',
            });
        }
    };

    return (
        <HeaderedContent
            title='Album Banners'
            description='Upload one or more banner images. They will cycle every 7 seconds on the public album page.'
            className='mt-16 pb-8'
        >
            <FlashMessageRender byKey='banner-error' />
            <BannerManager banners={banners} onAdd={handleAdd} onDelete={handleDelete} onReorder={handleReorder} />
        </HeaderedContent>
    );
}
