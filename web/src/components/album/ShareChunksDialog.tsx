import React from 'react';
import { Dialog, DialogActions, DialogBody, DialogDescription, DialogTitle } from '../elements/Dialog.tsx';
import { DescriptionList, DescriptionItem } from '../elements/DescriptionList.tsx';
import { Button } from '../elements/Button.tsx';
import { FileInfo } from '../../types.ts';
import { bytesToString } from '../../lib/formatters.ts';
import { BeakerIcon } from '@heroicons/react/16/solid';

export type ShareProgress = { current: number; total: number; size: number } | null;

interface ShareChunksDialogProps {
    open: boolean;
    onClose: (open: boolean) => void;
    images: FileInfo[];
    chunks: FileInfo[][];
    currentIndex: number;
    isSharing: boolean;
    progress: ShareProgress;
    onShareCurrent: () => Promise<void> | void;
    onNext: () => void;
}

const ShareChunksDialog: React.FC<ShareChunksDialogProps> = ({
    open,
    onClose,
    images,
    chunks,
    currentIndex,
    isSharing,
    progress,
    onShareCurrent,
    onNext,
}) => (
    <Dialog open={open} onClose={onClose}>
        <span
            className={
                'mb-2 inline-flex items-center rounded-md bg-indigo-50 px-2 py-1 text-xs font-medium text-indigo-700 ring-1 ring-indigo-700/10 ring-inset'
            }
        >
            <BeakerIcon className='my-auto mr-1 size-4' /> Experimental feature
        </span>
        <DialogTitle>Share Album in Multiple Parts</DialogTitle>
        <DialogDescription>
            This album includes {images.length} high-quality photos, totaling{' '}
            {bytesToString(images.reduce((sum, img) => sum + img.size, 0))}. Due to browser limitations, it will be
            shared in {chunks.length} parts, each up to 40MiB. Press "Share" to open your device’s share sheet, where
            you can send or save the images. After sharing each part, press "Next" to continue. This process may take
            some time due to the large file sizes.
        </DialogDescription>
        <DialogBody>
            <div className='space-y-4'>
                <DescriptionList>
                    <DescriptionItem term='Total Photos' details={images.length} />
                    <DescriptionItem
                        term='Total Size'
                        details={bytesToString(images.reduce((sum, img) => sum + img.size, 0))}
                    />
                    <DescriptionItem term='Number of Parts' details={chunks.length} />
                    <DescriptionItem term='Current Part' details={`${currentIndex + 1} of ${chunks.length}`} />
                </DescriptionList>

                {chunks[currentIndex] && (
                    <div className='mt-4'>
                        <h4 className='mb-2 text-sm font-medium text-gray-900 dark:text-white'>
                            Part {currentIndex + 1} Details:
                        </h4>
                        <DescriptionList>
                            <DescriptionItem term='Photos in this part' details={chunks[currentIndex].length} />
                            <DescriptionItem
                                term='Size of this part'
                                details={bytesToString(chunks[currentIndex].reduce((sum, img) => sum + img.size, 0))}
                            />
                        </DescriptionList>
                    </div>
                )}
            </div>
        </DialogBody>
        <DialogActions>
            <Button plain onClick={() => onClose(false)}>
                Cancel
            </Button>
            <Button onClick={onShareCurrent} disabled={isSharing}>
                {isSharing
                    ? progress
                        ? `Processing ${progress.current}/${progress.total} (${(progress.size / (1024 * 1024)).toFixed(1)}MB)`
                        : 'Sharing...'
                    : `Share Part ${currentIndex + 1}`}
            </Button>
            {currentIndex < chunks.length - 1 && <Button onClick={onNext}>Next Part</Button>}
        </DialogActions>
    </Dialog>
);

export default ShareChunksDialog;



