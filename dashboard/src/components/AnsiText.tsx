import React from 'react';

interface AnsiTextProps {
    text: string;
}

const colorMap: Record<string, string> = {
    '30': 'text-black',
    '31': 'text-red-500',
    '32': 'text-green-500',
    '33': 'text-yellow-500',
    '34': 'text-blue-500',
    '35': 'text-purple-500',
    '36': 'text-cyan-500',
    '37': 'text-gray-300', // Default white-ish
    '90': 'text-gray-500',
    '91': 'text-red-400',
    '92': 'text-green-400',
    '93': 'text-yellow-400',
    '94': 'text-blue-400',
    '95': 'text-purple-400',
    '96': 'text-cyan-400',
    '97': 'text-white',
};

export const AnsiText: React.FC<AnsiTextProps> = ({ text }) => {
    // Regex to split by ANSI escape sequences
    const parts = text.split(/(\u001b\[[0-9;]*m)/g);

    let currentColor = '';

    return (
        <>
            {parts.map((part, index) => {
                if (part.startsWith('\u001b[')) {
                    const codes = part.slice(2, -1).split(';');
                    codes.forEach(code => {
                        if (code === '0') {
                            currentColor = '';
                        } else if (colorMap[code]) {
                            currentColor = colorMap[code];
                        }
                    });
                    return null;
                }

                if (!part) return null;

                return (
                    <span
                        key={index}
                        className={currentColor}
                    >
                        {part}
                    </span>
                );
            })}
        </>
    );
};
