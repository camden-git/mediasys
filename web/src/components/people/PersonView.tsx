import React, { useEffect, useState, useCallback, useMemo } from 'react';
import { Link, useParams } from 'react-router-dom';
import { Person, FileInfo, PersonImageResult } from '../../types.ts';
import { getPersonById, getPersonImages, getPreviewImagePath } from '../../api.ts';
import AdvancedImageGrid from '../album/AdvancedImageGrid.tsx';
import ImageLightbox from '../album/ImageLightbox.tsx';
import LoadingSpinner from '../elements/LoadingSpinner.tsx';
import { UserCircleIcon, ArrowLeftIcon } from '@heroicons/react/24/outline';

const PAGE_SIZE = 60;

const PersonView: React.FC = () => {
    const { personId } = useParams<{ personId: string }>();
    const [person, setPerson] = useState<Person | null>(null);
    const [images, setImages] = useState<PersonImageResult[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [total, setTotal] = useState(0);
    const [hasMore, setHasMore] = useState(false);
    const [isLoadingMore, setIsLoadingMore] = useState(false);
    const [selectedImage, setSelectedImage] = useState<FileInfo | null>(null);

    useEffect(() => {
        if (!personId) return;
        const controller = new AbortController();
        setIsLoading(true);
        setError(null);

        (async () => {
            try {
                const personData = await getPersonById(Number(personId), controller.signal);
                setPerson(personData);
                const page = await getPersonImages(personData.id, { offset: 0, limit: PAGE_SIZE }, controller.signal);
                setImages(page.items);
                setTotal(page.total);
                setHasMore(page.has_more);
            } catch (err: any) {
                if (err.name !== 'AbortError') {
                    setError(err.message || 'Failed to load person data');
                }
            } finally {
                setIsLoading(false);
            }
        })();

        return () => controller.abort();
    }, [personId]);

    // Convert face search results to FileInfo objects for the grid, preferring the
    // lightweight thumbnail over the full-size preview when one is available.
    // thumbnail_path stays a relative path; consumers prefix the backend URL.
    const imageFiles: FileInfo[] = useMemo(
        () =>
            images.map(({ image_path: path, thumbnail_path, width, height }) => ({
                name: path.split('/').pop() ?? path,
                path: '/' + path,
                is_dir: false,
                size: 0,
                mod_time: 0,
                thumbnail_path: thumbnail_path || getPreviewImagePath(path),
                width,
                height,
            })),
        [images],
    );

    const selectedIndex = selectedImage ? imageFiles.findIndex((f) => f.path === selectedImage.path) : -1;

    const handleLoadMore = useCallback(async () => {
        if (!person) return;
        setIsLoadingMore(true);
        try {
            const page = await getPersonImages(person.id, { offset: images.length, limit: PAGE_SIZE });
            setImages((prev) => [...prev, ...page.items]);
            setTotal(page.total);
            setHasMore(page.has_more);
        } catch (err: any) {
            setError(err.message || 'Failed to load more photos');
        } finally {
            setIsLoadingMore(false);
        }
    }, [person, images.length]);

    const handleImageClick = useCallback((img: FileInfo) => {
        setSelectedImage(img);
    }, []);

    if (isLoading) return <LoadingSpinner />;
    if (error) return <p className='p-8 text-red-500'>{error}</p>;
    if (!person) return null;

    return (
        <div className='mx-auto max-w-6xl px-4 py-8'>
            {/* Header */}
            <div className='mb-6 flex items-center gap-4'>
                <Link
                    to='/people'
                    className='inline-flex items-center gap-1.5 text-sm text-gray-500 hover:text-gray-800 dark:text-gray-400 dark:hover:text-white'
                >
                    <ArrowLeftIcon className='size-4' />
                    All people
                </Link>
            </div>

            <div className='mb-8 flex items-center gap-4'>
                <UserCircleIcon className='size-16 text-gray-300 dark:text-gray-600' />
                <div>
                    <h1 className='text-3xl font-bold text-gray-950 dark:text-white'>{person.primary_name}</h1>
                    <p className='mt-1 text-sm text-gray-500 dark:text-gray-400'>
                        {total} photo{total !== 1 ? 's' : ''} found
                    </p>
                </div>
            </div>

            {imageFiles.length === 0 && (
                <p className='text-gray-500 dark:text-gray-400'>No photos found for this person yet.</p>
            )}

            {imageFiles.length > 0 && (
                <AdvancedImageGrid
                    images={imageFiles}
                    targetRowHeight={260}
                    boxSpacing={4}
                    onImageClick={handleImageClick}
                />
            )}

            {hasMore && (
                <div className='mt-6 flex justify-center'>
                    <button
                        type='button'
                        onClick={handleLoadMore}
                        disabled={isLoadingMore}
                        className='rounded-md px-4 py-2 text-sm font-medium text-gray-700 ring-1 ring-gray-300 hover:bg-gray-50 disabled:opacity-50 dark:text-gray-200 dark:ring-gray-600 dark:hover:bg-gray-800'
                    >
                        {isLoadingMore ? 'Loading...' : 'Load more'}
                    </button>
                </div>
            )}

            <ImageLightbox
                image={selectedImage}
                imageIndex={selectedIndex >= 0 ? selectedIndex : undefined}
                totalImages={imageFiles.length}
                onClose={() => setSelectedImage(null)}
                onPrev={() => {
                    if (selectedIndex > 0) setSelectedImage(imageFiles[selectedIndex - 1]);
                }}
                onNext={() => {
                    if (selectedIndex < imageFiles.length - 1) setSelectedImage(imageFiles[selectedIndex + 1]);
                }}
                canPrev={selectedIndex > 0}
                canNext={selectedIndex >= 0 && selectedIndex < imageFiles.length - 1}
            />
        </div>
    );
};

export default PersonView;
