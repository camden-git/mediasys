import React, { useEffect, useState, useCallback, useMemo } from 'react';
import { Link, useParams } from 'react-router-dom';
import { Person, FileInfo, PersonImageResult } from '../../types.ts';
import { getPersonById, searchFacesByName, getPreviewImagePath } from '../../api.ts';
import AdvancedImageGrid from '../album/AdvancedImageGrid.tsx';
import ImageLightbox from '../album/ImageLightbox.tsx';
import LoadingSpinner from '../elements/LoadingSpinner.tsx';
import { UserCircleIcon, ArrowLeftIcon } from '@heroicons/react/24/outline';

const PersonView: React.FC = () => {
    const { personId } = useParams<{ personId: string }>();
    const [person, setPerson] = useState<Person | null>(null);
    const [images, setImages] = useState<PersonImageResult[]>([]);
    const [isLoading, setIsLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
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
                const results = await searchFacesByName(personData.primary_name, controller.signal);
                setImages(results);
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
            images.map(({ image_path: path, thumbnail_path }) => ({
                name: path.split('/').pop() ?? path,
                path: '/' + path,
                is_dir: false,
                size: 0,
                mod_time: 0,
                thumbnail_path: thumbnail_path || getPreviewImagePath(path),
            })),
        [images],
    );

    const selectedIndex = selectedImage ? imageFiles.findIndex((f) => f.path === selectedImage.path) : -1;

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
                        {images.length} photo{images.length !== 1 ? 's' : ''} found
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
