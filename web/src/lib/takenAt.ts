// taken_at is the camera's wall-clock time encoded as UTC, so always format it in UTC.
const dateTimeFormat = new Intl.DateTimeFormat('en-US', { dateStyle: 'long', timeStyle: 'short', timeZone: 'UTC' });
const dateFormat = new Intl.DateTimeFormat('en-US', { dateStyle: 'medium', timeZone: 'UTC' });

export const formatTakenAt = (timestamp?: number): string | null => {
    if (!timestamp) return null;
    const date = new Date(timestamp * 1000);
    return Number.isNaN(date.getTime()) ? null : dateTimeFormat.format(date);
};

export const formatTakenAtDate = (timestamp?: number): string | null => {
    if (!timestamp) return null;
    const date = new Date(timestamp * 1000);
    return Number.isNaN(date.getTime()) ? null : dateFormat.format(date);
};
