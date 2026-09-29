import { FileInfo } from '../types';

export interface ProcessedRow {
    items: FileInfo[];
    height: number;
    rowIndex: number;
}

export interface LayoutOptions {
    containerWidth: number;
    targetRowHeight: number;
    boxSpacing: number;
    stretchLastRow?: boolean;
    maxRowHeightRatio?: number | null;
}

const getAspectRatio = (image: FileInfo): number => {
    if (!image.width || !image.height || image.height === 0) {
        return 1.5;
    }
    return image.width / image.height;
};

export const computeLayout = (images: FileInfo[], options: LayoutOptions): ProcessedRow[] => {
    const { containerWidth, targetRowHeight, boxSpacing, stretchLastRow = false, maxRowHeightRatio = 2.5 } = options;

    if (!containerWidth || containerWidth <= 0 || images.length === 0) {
        return [];
    }

    const processedRows: ProcessedRow[] = [];
    let currentRowItems: FileInfo[] = [];
    let currentRowAspectRatioSum = 0;
    let currentRowIndex = 0;

    const calculateHeight = (itemCount: number, arSum: number): number => {
        if (itemCount === 0 || arSum <= 0) return Infinity;
        const totalSpacing = Math.max(0, itemCount - 1) * boxSpacing;
        return (containerWidth - totalSpacing) / (arSum + Number.EPSILON);
    };

    const calculateCost = (height: number): number => {
        if (!isFinite(height)) return Infinity;
        return Math.abs(height - targetRowHeight);
    };

    const finalizeRow = (items: FileInfo[], arSum: number, isLastOverallRow: boolean): void => {
        if (items.length === 0) {
            return;
        }

        let finalHeight = calculateHeight(items.length, arSum);

        if (isLastOverallRow && !stretchLastRow) {
            finalHeight = Math.min(finalHeight, targetRowHeight);
        }

        if (maxRowHeightRatio && maxRowHeightRatio > 0) {
            const maxHeight = targetRowHeight * maxRowHeightRatio;
            if (finalHeight > maxHeight) {
                if (items.length > 1 || (items.length === 1 && getAspectRatio(items[0]) >= 1.0)) {
                    finalHeight = maxHeight;
                }
            }
        }

        finalHeight = Math.max(1, finalHeight);

        processedRows.push({ items, height: finalHeight, rowIndex: currentRowIndex });
        currentRowIndex++;
    };

    images.forEach((image) => {
        const aspectRatio = getAspectRatio(image);
        if (isNaN(aspectRatio) || aspectRatio <= 0) {
            console.warn(`[layout] skipping image ${image.name} due to invalid aspect ratio: ${aspectRatio}`);
            return;
        }

        const currentItems = currentRowItems;
        const currentARSum = currentRowAspectRatioSum;
        const currentHeight = calculateHeight(currentItems.length, currentARSum);
        const currentCost = calculateCost(currentHeight);

        const nextItems = [...currentItems, image];
        const nextARSum = currentARSum + aspectRatio;
        const nextHeight = calculateHeight(nextItems.length, nextARSum);
        const nextCost = calculateCost(nextHeight);

        let cutBeforeThisImage = false;

        if (currentItems.length > 0) {
            if (currentCost <= nextCost) {
                const singleItemBadFitThreshold = targetRowHeight * 0.75;
                if (!(currentItems.length === 1 && currentCost > singleItemBadFitThreshold)) {
                    cutBeforeThisImage = true;
                }
            } else {
                const minAllowedHeightThreshold = targetRowHeight * 0.65;
                if (nextHeight < minAllowedHeightThreshold) {
                    const currentHeightAcceptableFactor = 1.5;
                    if (currentHeight <= targetRowHeight * currentHeightAcceptableFactor) {
                        cutBeforeThisImage = true;
                    }
                }
            }
        }

        if (cutBeforeThisImage) {
            finalizeRow(currentItems, currentARSum, false);
            currentRowItems = [image];
            currentRowAspectRatioSum = aspectRatio;
        } else {
            currentRowItems = nextItems;
            currentRowAspectRatioSum = nextARSum;
        }
    });

    finalizeRow(currentRowItems, currentRowAspectRatioSum, true);

    return processedRows;
};
