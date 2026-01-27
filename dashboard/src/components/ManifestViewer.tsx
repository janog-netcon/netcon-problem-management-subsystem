import { useState } from 'react';
import yaml from 'js-yaml';
import { cleanManifest } from '../utils/manifest';
import { Card } from './Card';

interface ManifestViewerProps {
    problem: any;
    configMaps: { name: string, data: any }[];
}

export function ManifestViewer({ problem, configMaps }: ManifestViewerProps) {
    const [activeTab, setActiveTab] = useState('resource');

    const tabs = [
        { id: 'resource', label: 'Resource' },
        ...configMaps.map(cm => ({ id: `cm-${cm.name}`, label: `ConfigMap: ${cm.name}` }))
    ];

    const getActiveContent = () => {
        if (activeTab === 'resource') {
            return yaml.dump(cleanManifest(problem));
        }
        const cm = configMaps.find(c => `cm-${c.name}` === activeTab);
        if (cm && cm.data) {
            return yaml.dump(cleanManifest(cm.data));
        }
        return '';
    };

    return (
        <Card title="Resource Manifest">
            <div className="bg-gray-900 rounded-lg overflow-hidden font-mono text-xs">
                {/* Tabs */}
                <div className="flex border-b border-gray-700 overflow-x-auto">
                    {tabs.map(tab => (
                        <button
                            key={tab.id}
                            onClick={() => setActiveTab(tab.id)}
                            className={`px-4 py-2 font-medium focus:outline-none transition-colors whitespace-nowrap ${activeTab === tab.id
                                ? 'bg-gray-800 text-white border-b-2 border-indigo-500'
                                : 'text-gray-400 hover:text-gray-200 hover:bg-gray-800/50'
                                }`}
                        >
                            {tab.label}
                        </button>
                    ))}
                </div>

                {/* Content */}
                <div className="max-h-[600px] overflow-auto p-4">
                    <pre className="text-gray-300 whitespace-pre font-mono leading-relaxed">
                        {getActiveContent()}
                    </pre>
                </div>
            </div>
        </Card>
    );
}
