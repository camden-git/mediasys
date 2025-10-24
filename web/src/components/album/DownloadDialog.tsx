import React from 'react';
import { Dialog, DialogActions, DialogBody, DialogDescription, DialogTitle } from '../elements/Dialog.tsx';
import { Button } from '../elements/Button.tsx';
import { bytesToString } from '../../lib/formatters.ts';

interface DownloadDialogProps {
    open: boolean;
    onClose: (open: boolean) => void;
    albumName?: string;
    zipSize?: number;
    onDownload: () => void;
}

const DownloadDialog: React.FC<DownloadDialogProps> = ({ open, onClose, albumName, zipSize, onDownload }) => (
    <Dialog open={open} onClose={onClose}>
        <DialogTitle>Download {albumName}</DialogTitle>
        <DialogDescription>
            A {zipSize ? bytesToString(zipSize) : ''} zip file containing all images in this album is available for
            download. This download may take a long time as the images are in the highest quality. Individual photos can
            be downloaded by opening an image and pressing the download icon in the top right.
        </DialogDescription>
        <DialogBody></DialogBody>
        <DialogActions>
            <Button plain onClick={() => onClose(false)}>
                Cancel
            </Button>
            <Button onClick={onDownload}>Download</Button>
        </DialogActions>
    </Dialog>
);

export default DownloadDialog;



