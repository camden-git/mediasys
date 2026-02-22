import React, { useEffect, useState } from 'react';
import { Heading } from '../../../elements/Heading';
import { Text } from '../../../elements/Text';
import PageContentBlock from '../../../elements/PageContentBlock.tsx';
import LoadingSpinner from '../../../elements/LoadingSpinner';
import { UntaggedFaceResult, Person } from '../../../../types';
import { getUntaggedFaces, getPeople } from '../../../../api';
import { useStoreState } from '../../../../store/hooks';
import FaceThumbnail from '../../faces/shared/FaceThumbnail';
import FaceLightboxModal from '../../faces/shared/FaceLightboxModal';

const AlbumFaceTaggingContainer: React.FC = () => {
    const album = useStoreState((state) => state.albumContext.data);

    const [allFaces, setAllFaces] = useState<UntaggedFaceResult[]>([]);
    const [faces, setFaces] = useState<UntaggedFaceResult[]>([]);
    const [people, setPeople] = useState<Person[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [lightboxIndex, setLightboxIndex] = useState<number | null>(null);

    useEffect(() => {
        setLoading(true);
        setError(null);
        Promise.all([getUntaggedFaces({ limit: 100 }), getPeople()])
            .then(([f, p]) => {
                setAllFaces(f);
                setPeople(p);
            })
            .catch((e) => setError(e.message))
            .finally(() => setLoading(false));
    }, []);

    // Filter faces to this album's folder path
    useEffect(() => {
        if (!album) return;
        setFaces(allFaces.filter((f) => f.image_path.startsWith(album.folder_path)));
    }, [allFaces, album]);

    const handleTagged = (faceId: number) => {
        setAllFaces((prev) => {
            const next = prev.filter((f) => f.face_id !== faceId);
            // faces state will update via useEffect above
            // advance or close lightbox based on next filtered list
            const nextFiltered = album ? next.filter((f) => f.image_path.startsWith(album.folder_path)) : next;
            setLightboxIndex((idx) => {
                if (idx === null) return null;
                if (nextFiltered.length === 0) return null;
                return Math.min(idx, nextFiltered.length - 1);
            });
            return next;
        });
    };

    const handleDeleted = (faceId: number) => handleTagged(faceId);

    const handlePersonCreated = (person: Person) => setPeople((prev) => [...prev, person]);

    const handleSuggestionUpdated = (
        faceId: number,
        personId: number | null,
        personName: string | null,
        suggestionCount: number,
    ) => {
        setAllFaces((prev) =>
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
                                    imagePath={face.image_path}
                                    x1={face.x1}
                                    y1={face.y1}
                                    x2={face.x2}
                                    y2={face.y2}
                                    imageWidth={face.image_width}
                                    imageHeight={face.image_height}
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
                    people={people}
                    onClose={() => setLightboxIndex(null)}
                    onNavigate={setLightboxIndex}
                    onTagged={handleTagged}
                    onDeleted={handleDeleted}
                    onPersonCreated={handlePersonCreated}
                    onSuggestionUpdated={handleSuggestionUpdated}
                />
            )}
        </PageContentBlock>
    );
};

export default AlbumFaceTaggingContainer;
