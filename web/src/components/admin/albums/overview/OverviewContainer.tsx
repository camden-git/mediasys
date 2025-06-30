import React from 'react';
import { useStoreState } from '../../../../store/hooks.ts';
import { getBannerUrl } from '../../../../api.ts';
import { Heading } from '../../../elements/Heading.tsx';
import { CameraIcon, MapPinIcon, PhotoIcon } from '@heroicons/react/16/solid';

const OverviewContainer: React.FC = () => {
    const album = useStoreState((state) => state.albumContext.data!);

    return (
        <div className='relative mx-auto'>
            <div className='absolute inset-x-0 top-0 -z-10 h-80 overflow-hidden rounded-t-2xl mask-b-from-60% sm:h-88 md:h-112 lg:h-128'>
                {album.banner_image_path && (
                    <img
                        alt=''
                        src={getBannerUrl(album.banner_image_path)}
                        className='absolute inset-0 h-full w-full mask-l-from-60% object-cover object-center opacity-40'
                    />
                )}
                <div className='absolute inset-0 rounded-t-2xl outline-1 -outline-offset-1 outline-gray-950/10 dark:outline-white/10' />
            </div>
            <div className='mx-auto'>
                <div className='relative'>
                    <div className='px-8 pt-48 pb-12 lg:py-24'>
                        {/*<Logo className="h-8 fill-gray-950 dark:fill-white" />*/}
                        <h1 className='sr-only'>{album.name} overview</h1>
                        <Heading className={'truncate font-bold'} huge>
                            {album.name}
                        </Heading>
                        <p className='mt-7 max-w-lg text-base/7 text-pretty text-gray-600 dark:text-gray-400'>
                            {album.description}
                        </p>
                        <div className='mt-6 flex flex-wrap items-center gap-x-4 gap-y-3 text-sm/7 font-semibold text-gray-950 sm:gap-3'>
                            <div className='flex items-center gap-1.5'>
                                <PhotoIcon className='size-4 text-gray-950/40' />
                            </div>
                            <span className='hidden text-gray-950/25 sm:inline dark:text-white/25'>&middot;</span>
                            <div className='flex items-center gap-1.5'>
                                <CameraIcon className='size-4 text-gray-950/40' />
                                Camden Rush
                            </div>
                            {album.location && (
                                <>
                                    <span className='hidden text-gray-950/25 sm:inline dark:text-white/25'>
                                        &middot;
                                    </span>
                                    <div className='flex items-center gap-1.5'>
                                        <MapPinIcon className='size-4 text-gray-950/40' />
                                        {album.location}
                                    </div>
                                </>
                            )}
                        </div>
                    </div>

                    <div className='mt-4'>drop</div>
                </div>
            </div>
        </div>
    );
};

export default OverviewContainer;
