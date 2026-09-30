import React, { useEffect, useState } from 'react';
import { Heading } from '../../../elements/Heading';
import { Text } from '../../../elements/Text';
import PageContentBlock from '../../../elements/PageContentBlock.tsx';
import LoadingSpinner from '../../../elements/LoadingSpinner';
import { UntaggedFaceResult } from '../../../../types';
import { getUntaggedFaces } from '../../../../api/faces';
import { errorMessage, isAbortError } from '../../../../api/errors';
import { useAlbumData } from '../../../../store/albumContextHooks';
import FaceThumbnail from '../../faces/shared/FaceThumbnail';
import FaceLightboxModal from '../../faces/shared/FaceLightboxModal';

const AlbumFaceTaggingContainer: React.FC = () => {
    const album = useAlbumData();

    const [faces, setFaces] = useState<UntaggedFaceResult[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [lightboxIndex, setLightboxIndex] = useState<number | null>(null);

    const albumId = album?.id;

    useEffect(() => {
        if (albumId == null) return;
        const ctrl = new AbortController();
        setLoading(true);
        setError(null);
        getUntaggedFaces({ limit: 100, album_id: albumId }, ctrl.signal)
            .then(setFaces)
            .catch((e) => {
                if (!isAbortError(e)) setError(errorMessage(e, 'Failed to load faces'));
            })
            .finally(() => {
                if (!ctrl.signal.aborted) setLoading(false);
            });
        return () => ctrl.abort();
    }, [albumId]);

    const handleTagged = (faceId: number) => {
        setFaces((prev) => {
            const next = prev.filter((f) => f.face_id !== faceId);
            setLightboxIndex((idx) => {
                if (idx === null) return null;
                if (next.length === 0) return null;
                return Math.min(idx, next.length - 1);
            });
            return next;
        });
    };

    const handleDeleted = (faceId: number) => handleTagged(faceId);

    const handleSuggestionUpdated = (
        faceId: number,
        personId: number | null,
        personName: string | null,
        suggestionCount: number,
    ) => {
        setFaces((prev) =>
            prev.map((f) =>
                f.face_id === faceId
                    ? {
                          ...f,
                          suggested_person_id: personId,
                          suggested_person_name: personName,
                          suggestion_count: suggestionCount,
                      }
                    : f,
            ),
        );
    };

    if (loading) return <LoadingSpinner />;

    return (
        <PageContentBlock title='Face Tagging'>
            <div className='mb-6'>
                <Heading>Face Tagging</Heading>
                <Text className='mt-1'>Review untagged faces in this album and assign them to people.</Text>
            </div>

            {error && <p className='mb-4 text-sm text-red-600'>{error}</p>}

            {faces.length === 0 ? (
                <div className='flex flex-col items-center justify-center rounded-lg border border-dashed border-zinc-300 p-16 text-center'>
                    <Heading level={4}>All caught up!</Heading>
                    <Text className='mt-2 text-sm text-zinc-500'>No untagged faces in this album.</Text>
                </div>
            ) : (
                <>
                    <p className='mb-4 text-sm text-zinc-500'>
                        {faces.length} face{faces.length !== 1 ? 's' : ''} to review
                    </p>
                    <div className='grid grid-cols-2 gap-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5'>
                        {faces.map((face, i) => (
                            <div
                                key={face.face_id}
                                className='group relative overflow-hidden rounded-lg border border-zinc-200 dark:border-zinc-700'
                            >
                                <FaceThumbnail
                                    faceId={face.face_id}
                                    label={`Review face ${i + 1}`}
                                    onClick={() => setLightboxIndex(i)}
                                />
                                <span className='absolute top-1.5 right-1.5 rounded-full bg-black/60 px-1.5 py-0.5 text-[10px] text-white'>
                                    {Math.round(face.detection_confidence * 100)}%
                                </span>
                                {face.suggested_person_name && (
                                    <span className='absolute bottom-1.5 left-1.5 max-w-[80%] truncate rounded bg-indigo-600/80 px-1.5 py-0.5 text-[10px] text-white'>
                                        {face.suggested_person_name}
                                    </span>
                                )}
                            </div>
                        ))}
                    </div>
                </>
            )}

            {lightboxIndex !== null && (
                <FaceLightboxModal
                    faces={faces}
                    currentIndex={lightboxIndex}
                    onClose={() => setLightboxIndex(null)}
                    onNavigate={setLightboxIndex}
                    onTagged={handleTagged}
                    onDeleted={handleDeleted}
                    onSuggestionUpdated={handleSuggestionUpdated}
                />
            )}
        </PageContentBlock>
    );
};

export default AlbumFaceTaggingContainer;
