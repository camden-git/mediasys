import { useAlbumContextStore } from '../../../../store/useAlbumContextStore';
import { useUIStore } from '../../../../store/useUIStore';
import HeaderedContent from '../../../elements/HeaderedContent.tsx';
import { addAlbumBanner, deleteAlbumBanner, getAlbum, reorderAlbumBanners } from '../../../../api/admin/albums';
import { BannerManager } from '../../shared/BannerManager';

export function BannerUpload() {
    const albumId = useAlbumContextStore((s) => s.data!.id);
    const banners = useAlbumContextStore((s) => s.data?.banners ?? []);
    const addFlash = useUIStore((s) => s.addFlash);
    const setAlbum = useAlbumContextStore((s) => s.setAlbum);

    const refreshAlbum = async () => {
        setAlbum(await getAlbum(albumId));
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
                message: error.response?.data?.error || 'Failed to add banner',
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
                message: error.response?.data?.error || 'Failed to remove banner',
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
                message: error.response?.data?.error || 'Failed to reorder banners',
            });
        }
    };

    return (
        <HeaderedContent
            title='Album Banners'
            description='Upload one or more banner images. They will cycle every 7 seconds on the public album page.'
            className='mt-16 pb-8'
        >
            <BannerManager banners={banners} onAdd={handleAdd} onDelete={handleDelete} onReorder={handleReorder} />
        </HeaderedContent>
    );
}
