const _CONVERSION_UNIT = 1024;

/**
 * Given a number of bytes, converts them into a human-readable string format
 * using "1024" as the divisor
 */
function bytesToString(bytes: number, decimals = 2): string {
    const k = _CONVERSION_UNIT;

    if (bytes < 1) return '0 Bytes';

    decimals = Math.floor(Math.max(0, decimals));
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    const value = Number((bytes / Math.pow(k, i)).toFixed(decimals));

    return `${value} ${['Bytes', 'KiB', 'MiB', 'GiB', 'TiB'][i]}`;
}

/**
 * Formats a date string to a human-readable format
 */
function formatDate(dateString: string): string {
    const date = new Date(dateString);
    return date.toLocaleDateString('en-US', {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
    });
}

/**
 * Calculates time remaining until a given date
 */
function getTimeRemaining(targetDate: string): {
    total: number;
    days: number;
    hours: number;
    minutes: number;
    seconds: number;
    isExpired: boolean;
} {
    const now = new Date().getTime();
    const target = new Date(targetDate).getTime();
    const total = target - now;

    if (total <= 0) {
        return {
            total: 0,
            days: 0,
            hours: 0,
            minutes: 0,
            seconds: 0,
            isExpired: true,
        };
    }

    const days = Math.floor(total / (1000 * 60 * 60 * 24));
    const hours = Math.floor((total % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60));
    const minutes = Math.floor((total % (1000 * 60 * 60)) / (1000 * 60));
    const seconds = Math.floor((total % (1000 * 60)) / 1000);

    return {
        total,
        days,
        hours,
        minutes,
        seconds,
        isExpired: false,
    };
}

/**
 * Formats time remaining as a human-readable string
 */
function formatTimeRemaining(timeRemaining: ReturnType<typeof getTimeRemaining>): string {
    if (timeRemaining.isExpired) {
        return 'Expired';
    }

    const { days, hours, minutes, seconds } = timeRemaining;

    if (days > 0) {
        return `${days}d ${hours}h ${minutes}m`;
    } else if (hours > 0) {
        return `${hours}h ${minutes}m ${seconds}s`;
    } else if (minutes > 0) {
        return `${minutes}m ${seconds}s`;
    } else {
        return `${seconds}s`;
    }
}

/**
 * Determines the urgency level of a date
 */
function getUrgencyLevel(targetDate: string): 'normal' | 'warning' | 'critical' {
    const timeRemaining = getTimeRemaining(targetDate);
    
    if (timeRemaining.isExpired) {
        return 'critical';
    }

    const totalHours = timeRemaining.total / (1000 * 60 * 60);
    
    if (totalHours < 1) {
        return 'critical';
    } else if (totalHours < 24) {
        return 'warning';
    }
    
    return 'normal';
}

export { bytesToString, formatDate, getTimeRemaining, formatTimeRemaining, getUrgencyLevel };
