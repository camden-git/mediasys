import React from 'react';
import CountdownTimer from './CountdownTimer';
import {formatDate, getUrgencyLevel} from "../../lib/formatters.ts";

interface DateDisplayProps {
    dateString: string;
    showCountdown?: boolean;
    className?: string;
}

const DateDisplay: React.FC<DateDisplayProps> = ({ 
    dateString, 
    showCountdown = false, 
    className = '' 
}) => {
    const urgencyLevel = getUrgencyLevel(dateString);
    
    const getUrgencyClasses = () => {
        switch (urgencyLevel) {
            case 'critical':
                return 'text-red-600';
            case 'warning':
                return 'text-yellow-600';
            default:
                return 'text-zinc-500';
        }
    };

    return (
        <div className={`${className}`}>
            <div className={getUrgencyClasses()}>
                {formatDate(dateString)}
            </div>
            {showCountdown && (
                <div className="text-sm mt-1">
                    <CountdownTimer targetDate={dateString} />
                </div>
            )}
        </div>
    );
};

export default DateDisplay;






