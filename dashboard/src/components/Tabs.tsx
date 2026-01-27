import { useState, ReactNode } from 'react';

interface Tab {
    id: string;
    label: string;
    content: ReactNode;
    icon?: ReactNode;
}

interface TabsProps {
    tabs: Tab[];
    defaultTabId?: string;
}

export function Tabs({ tabs, defaultTabId }: TabsProps) {
    const [activeTabId, setActiveTabId] = useState(defaultTabId || tabs[0]?.id);

    const activeTab = tabs.find((t) => t.id === activeTabId);

    return (
        <div>
            <div className="border-b border-gray-200 dark:border-gray-700">
                <nav className="-mb-px flex space-x-8" aria-label="Tabs">
                    {tabs.map((tab) => (
                        <button
                            key={tab.id}
                            onClick={() => setActiveTabId(tab.id)}
                            className={`
                whitespace-nowrap py-4 px-1 border-b-2 font-medium text-sm flex items-center
                ${activeTabId === tab.id
                                    ? 'border-indigo-500 text-indigo-600 dark:text-indigo-400'
                                    : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300 dark:text-gray-400 dark:hover:text-gray-300'
                                }
              `}
                            aria-current={activeTabId === tab.id ? 'page' : undefined}
                        >
                            {tab.icon && <span className="mr-2">{tab.icon}</span>}
                            {tab.label}
                        </button>
                    ))}
                </nav>
            </div>
            <div className="mt-6">
                {activeTab?.content}
            </div>
        </div>
    );
}
