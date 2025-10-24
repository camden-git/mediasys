import React, { useState, useEffect } from 'react';
import { getTimeRemaining, formatTimeRemaining, getUrgencyLevel } from '../../lib/formatters';

interface CountdownTimerProps {
    targetDate: string;
    className?: string;
}

const CountdownTimer: React.FC<CountdownTimerProps> = ({ targetDate, className = '' }) => {
    const [timeRemaining, setTimeRemaining] = useState(getTimeRemaining(targetDate));
    const urgencyLevel = getUrgencyLevel(targetDate);

    useEffect(() => {
        const timer = setInterval(() => {
            setTimeRemaining(getTimeRemaining(targetDate));
        }, 1000);

        return () => clearInterval(timer);
    }, [targetDate]);

    const getClassName = () => {
        const baseClasses = 'font-medium transition-colors duration-200';
        
        switch (urgencyLevel) {
            case 'critical':
                return `${baseClasses} text-red-600 ${timeRemaining.total > 0 ? 'animate-pulse' : ''}`;
            case 'warning':
                return `${baseClasses} text-yellow-600`;
            default:
                return `${baseClasses} text-zinc-500`;
        }
    };

    return (
        <span className={`${getClassName()} ${className}`}>
            {formatTimeRemaining(timeRemaining)}
        </span>
    );
};

export default CountdownTimer;






