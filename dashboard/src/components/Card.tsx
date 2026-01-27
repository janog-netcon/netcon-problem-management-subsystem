import { ReactNode } from 'react';

interface CardProps {
    title?: ReactNode;
    children: ReactNode;
    className?: string;
    action?: ReactNode;
}

export function Card({ title, children, className = '', action }: CardProps) {
    return (
        <div className={`bg-white dark:bg-gray-800 shadow rounded-lg overflow-hidden ${className}`}>
            {(title || action) && (
                <div className="px-6 py-5 border-b border-gray-200 dark:border-gray-700 flex justify-between items-center">
                    {title && (
                        <h3 className="text-lg leading-6 font-medium text-gray-900 dark:text-white flex items-center">
                            {title}
                        </h3>
                    )}
                    {action && <div>{action}</div>}
                </div>
            )}
            <div className="px-6 py-5">
                {children}
            </div>
        </div>
    );
}
