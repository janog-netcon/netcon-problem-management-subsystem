import { useState, useRef, useEffect } from 'react';
import { ChevronDown, Search, X, Check } from 'lucide-react';

interface Option {
    label: string;
    value: string;
}

interface MultiSelectProps {
    label: string;
    options: Option[];
    selectedValues: string[];
    onChange: (value: string) => void;
    placeholder?: string;
}

export function MultiSelect({
    label,
    options,
    selectedValues,
    onChange,
    placeholder = 'Select options...'
}: MultiSelectProps) {
    const [isOpen, setIsOpen] = useState(false);
    const [searchTerm, setSearchTerm] = useState('');
    const dropdownRef = useRef<HTMLDivElement>(null);

    useEffect(() => {
        const handleClickOutside = (event: MouseEvent) => {
            if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
                setIsOpen(false);
            }
        };

        document.addEventListener('mousedown', handleClickOutside);
        return () => document.removeEventListener('mousedown', handleClickOutside);
    }, []);

    const filteredOptions = options.filter(option =>
        option.label.toLowerCase().includes(searchTerm.toLowerCase())
    );

    return (
        <div className="relative inline-block w-full sm:w-64" ref={dropdownRef}>
            <label className="block text-xs font-semibold text-gray-500 uppercase tracking-wider dark:text-gray-400 mb-1">
                {label}
            </label>
            <button
                type="button"
                onClick={() => setIsOpen(!isOpen)}
                className="relative w-full bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-700 rounded-md shadow-sm pl-3 pr-10 py-2 text-left cursor-default focus:outline-none focus:ring-1 focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm"
            >
                <span className="block truncate text-gray-700 dark:text-gray-200">
                    {selectedValues.length > 0
                        ? `${selectedValues.length} selected`
                        : placeholder}
                </span>
                <span className="absolute inset-y-0 right-0 flex items-center pr-2 pointer-events-none">
                    <ChevronDown className="h-4 w-4 text-gray-400" />
                </span>
            </button>

            {isOpen && (
                <div className="absolute z-10 mt-1 w-full bg-white dark:bg-gray-800 shadow-lg max-h-60 rounded-md py-1 text-base ring-1 ring-black ring-opacity-5 overflow-auto focus:outline-none sm:text-sm">
                    <div className="sticky top-0 z-10 bg-white dark:bg-gray-800 px-2 py-1.5 border-b border-gray-200 dark:border-gray-700">
                        <div className="relative">
                            <div className="absolute inset-y-0 left-0 pl-2.5 flex items-center pointer-events-none">
                                <Search className="h-4 w-4 text-gray-400" />
                            </div>
                            <input
                                type="text"
                                className="block w-full pl-9 pr-3 py-1.5 border border-gray-300 dark:border-gray-600 rounded-md leading-5 bg-white dark:bg-gray-700 placeholder-gray-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm"
                                placeholder="Filter options..."
                                value={searchTerm}
                                onChange={(e) => setSearchTerm(e.target.value)}
                            />
                            {searchTerm && (
                                <button
                                    onClick={() => setSearchTerm('')}
                                    className="absolute inset-y-0 right-0 pr-2.5 flex items-center"
                                >
                                    <X className="h-4 w-4 text-gray-400 hover:text-gray-600" />
                                </button>
                            )}
                        </div>
                    </div>
                    <ul className="max-h-48 overflow-auto py-1">
                        {filteredOptions.length > 0 ? (
                            filteredOptions.map((option) => (
                                <li
                                    key={option.value}
                                    className="relative cursor-default select-none py-2 pl-3 pr-9 hover:bg-indigo-600 hover:text-white dark:hover:bg-indigo-600 group"
                                    onClick={() => onChange(option.value)}
                                >
                                    <div className="flex items-center">
                                        <div className={`flex shrink-0 items-center justify-center h-4 w-4 rounded border ${selectedValues.includes(option.value)
                                                ? 'bg-indigo-500 border-indigo-500'
                                                : 'bg-white dark:bg-gray-700 border-gray-300 dark:border-gray-600'
                                            } transition-colors mr-2 group-hover:border-white`}>
                                            {selectedValues.includes(option.value) && (
                                                <Check className="h-3 w-3 text-white" />
                                            )}
                                        </div>
                                        <span className={`block truncate ${selectedValues.includes(option.value) ? 'font-semibold' : 'font-normal'}`}>
                                            {option.label}
                                        </span>
                                    </div>
                                </li>
                            ))
                        ) : (
                            <li className="text-gray-500 dark:text-gray-400 italic py-2 px-3 text-center">
                                No results found
                            </li>
                        )}
                    </ul>
                </div>
            )}
        </div>
    );
}
