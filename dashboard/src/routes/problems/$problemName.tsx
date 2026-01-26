import { createFileRoute, Link } from '@tanstack/react-router';
import { useState } from 'react';
import { getProblem, getProblemEnvironments, getWorkers, getConfigMap } from '../../data/k8s';
import yaml from 'js-yaml';
import { cleanManifest } from '../../utils/manifest';
import { ChevronLeft, Box, Activity, Network, FileCode, ChevronDown, LineChart } from 'lucide-react';
import { Tabs } from '../../components/Tabs';
import { Card } from '../../components/Card';

export const Route = createFileRoute('/problems/$problemName')({
    component: ProblemDetailPage,
    loader: async ({ params }) => {
        const [problem, envsList, workers] = await Promise.all([
            getProblem({ data: params.problemName }),
            getProblemEnvironments(),
            getWorkers(),
        ]);

        const cmNames = new Set<string>();
        if (problem.spec.template?.spec?.topologyFile?.configMapRef?.name) {
            cmNames.add(problem.spec.template.spec.topologyFile.configMapRef.name);
        }
        if (problem.spec.template?.spec?.configFiles) {
            problem.spec.template.spec.configFiles.forEach((cf: any) => {
                if (cf.configMapRef?.name) {
                    cmNames.add(cf.configMapRef.name);
                }
            });
        }

        const configMaps = await Promise.all(
            Array.from(cmNames).map(async name => {
                const cm = await getConfigMap({ data: name });
                return { name, data: cm };
            })
        );

        return { problem, envsList, workers, configMaps };
    },
});

function ProblemDetailPage() {
    const { problem, envsList, workers, configMaps } = Route.useLoaderData();
    const [isActionMenuOpen, setIsActionMenuOpen] = useState(false);

    const relatedEnvs = envsList.items.filter((env) => {
        return env.metadata.ownerReferences?.some(
            (ref) => ref.kind === 'Problem' && ref.name === problem.metadata.name
        );
    });

    // Calculate stats

    const tabs = [
        {
            id: 'overview',
            label: 'Overview',
            icon: <Activity className="w-4 h-4" />,
            content: (
                <div className="space-y-6">
                    <div className="grid grid-cols-1 md:grid-cols-5 gap-6">
                        <StatusCard label="Desired" value={problem.spec.assignableReplicas} color="gray" />
                        <StatusCard label="Ready" value={problem.status?.replicas?.assignable ?? 0} color="green" />
                        <StatusCard label="Assigned" value={problem.status?.replicas?.assigned ?? 0} color="indigo" />
                        <StatusCard label="Deploying" value={(problem.status?.replicas?.total ?? 0) - ((problem.status?.replicas?.assignable ?? 0) + (problem.status?.replicas?.assigned ?? 0))} color="blue" />
                        <StatusCard label="Total" value={problem.status?.replicas?.total ?? 0} color="purple" />
                    </div>



                    <Card title="Worker Distribution">
                        <div className="overflow-x-auto mt-2">
                            <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
                                <thead className="bg-gray-50 dark:bg-gray-700">
                                    <tr>
                                        <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Worker</th>
                                        <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Ready</th>
                                        <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Assigned</th>
                                        <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Deploying</th>
                                        <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Total</th>
                                        <th scope="col" className="relative px-6 py-3"><span className="sr-only">View</span></th>
                                    </tr>
                                </thead>
                                <tbody className="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
                                    {workers.items.map(worker => {
                                        const workerEnvs = relatedEnvs.filter(e => e.spec.workerName === worker.metadata.name);
                                        if (workerEnvs.length === 0) return null;

                                        const total = workerEnvs.length;
                                        const assigned = workerEnvs.filter(e => e.status?.conditions?.some(c => c.type === 'Assigned' && c.status === 'True')).length;
                                        const ready = workerEnvs.filter(e => e.status?.conditions?.some(c => c.type === 'Ready' && c.status === 'True')).length - assigned;
                                        const deploying = total - (ready + assigned);

                                        return (
                                            <tr key={worker.metadata.name} className="hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors">
                                                <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900 dark:text-white">
                                                    <div className="flex items-center">
                                                        <Network className="w-4 h-4 mr-2 text-gray-400" />
                                                        {worker.metadata.name}
                                                    </div>
                                                </td>
                                                <td className="px-6 py-4 whitespace-nowrap text-sm text-green-600 dark:text-green-400">
                                                    {ready}
                                                </td>
                                                <td className="px-6 py-4 whitespace-nowrap text-sm text-indigo-600 dark:text-indigo-400">
                                                    {assigned}
                                                </td>
                                                <td className="px-6 py-4 whitespace-nowrap text-sm text-blue-600 dark:text-blue-400">
                                                    {deploying}
                                                </td>
                                                <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900 dark:text-white">
                                                    {total}
                                                </td>
                                                <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                                                    <Link
                                                        to="/problem-environments"
                                                        search={{
                                                            problem: [problem.metadata.name],
                                                            worker: [worker.metadata.name]
                                                        }}
                                                        className="text-indigo-600 hover:text-indigo-900 dark:text-indigo-400 dark:hover:text-indigo-300"
                                                    >
                                                        View List
                                                    </Link>
                                                </td>
                                            </tr>
                                        );
                                    })}
                                    {relatedEnvs.some(e => !e.spec.workerName) && (
                                        <tr className="bg-gray-50/30 dark:bg-gray-900/30">
                                            <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-500 dark:text-gray-400 italic">
                                                Unscheduled
                                            </td>
                                            <td colSpan={4} className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                                                {relatedEnvs.filter(e => !e.spec.workerName).length}
                                            </td>
                                            <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                                                <Link
                                                    to="/problem-environments"
                                                    search={{
                                                        problem: [problem.metadata.name],
                                                        worker: [""]
                                                    }}
                                                    className="text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300"
                                                >
                                                    View List
                                                </Link>
                                            </td>
                                        </tr>
                                    )}
                                </tbody>
                            </table>
                        </div>
                    </Card>
                </div>
            ),
        },

        {

            id: 'manifest',
            label: 'Manifests',
            icon: <FileCode className="w-4 h-4" />,
            content: (
                <div className="space-y-6">
                    <ManifestViewer problem={problem} configMaps={configMaps} />
                </div>
            )
        }

    ];

    return (
        <div className="p-6 bg-gray-50 dark:bg-gray-900 min-h-screen">
            <div className="max-w-7xl mx-auto space-y-6">
                {/* Header */}
                <div>
                    <Link to="/problems" className="inline-flex items-center text-sm text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 mb-4 transition-colors">
                        <ChevronLeft className="w-4 h-4 mr-1" />
                        Back to Problems
                    </Link>
                    <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4 w-full">
                        <div className="flex items-center space-x-3">
                            <div className="p-3 bg-indigo-100 dark:bg-indigo-900/50 rounded-lg">
                                <Box className="w-8 h-8 text-indigo-600 dark:text-indigo-400" />
                            </div>
                            <div>
                                <h1 className="text-3xl font-bold text-gray-900 dark:text-white">{problem.metadata.name}</h1>

                            </div>
                        </div>
                        <div className="flex items-center">
                            {/* Actions Dropdown */}
                            <div className="relative">
                                <button
                                    onClick={() => setIsActionMenuOpen(!isActionMenuOpen)}
                                    className="inline-flex items-center px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm text-sm font-medium text-gray-700 dark:text-gray-200 bg-white dark:bg-gray-800 hover:bg-gray-50 dark:hover:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 transition-colors"
                                >
                                    Actions
                                    <ChevronDown className="ml-2 -mr-1 h-4 w-4" />
                                </button>

                                {isActionMenuOpen && (
                                    <>
                                        <div className="fixed inset-0 z-10" onClick={() => setIsActionMenuOpen(false)}></div>
                                        <div className="origin-top-right absolute right-0 mt-2 w-56 rounded-md shadow-lg bg-white dark:bg-gray-800 ring-1 ring-black ring-opacity-5 z-20 focus:outline-none">
                                            <div className="py-1">
                                                <a
                                                    href={`https://janog57-grafana.proelbtn.com/d/admqpqx/telescope-3a-3a-problem?var-name=${problem.metadata.name}`}
                                                    target="_blank"
                                                    rel="noopener noreferrer"
                                                    className="flex items-center w-full px-4 py-2 text-sm text-left text-gray-700 dark:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700"
                                                    onClick={() => setIsActionMenuOpen(false)}
                                                >
                                                    <LineChart className="mr-3 h-4 w-4" />
                                                    Grafana
                                                </a>
                                                <Link
                                                    to="/problem-environments"
                                                    search={{ problem: [problem.metadata.name] }}
                                                    className="flex items-center w-full px-4 py-2 text-sm text-left text-gray-700 dark:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700"
                                                    onClick={() => setIsActionMenuOpen(false)}
                                                >
                                                    <Activity className="mr-3 h-4 w-4" />
                                                    Search Environments
                                                </Link>
                                            </div>
                                        </div>
                                    </>
                                )}
                            </div>
                        </div>
                    </div>
                </div>

                <Tabs tabs={tabs} />
            </div>
        </div>
    );
}


function StatusCard({ label, value, color }: { label: string, value: number, color: string }) {
    const colorClasses: Record<string, string> = {
        gray: 'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200 border-gray-200 dark:border-gray-700',
        blue: 'bg-blue-50 text-blue-800 dark:bg-blue-900/30 dark:text-blue-200 border-blue-200 dark:border-blue-900',
        green: 'bg-green-50 text-green-800 dark:bg-green-900/30 dark:text-green-200 border-green-200 dark:border-green-900',
        purple: 'bg-purple-50 text-purple-800 dark:bg-purple-900/30 dark:text-purple-200 border-purple-200 dark:border-purple-900',
        indigo: 'bg-indigo-50 text-indigo-800 dark:bg-indigo-900/30 dark:text-indigo-200 border-indigo-200 dark:border-indigo-900',
    };

    return (
        <div className={`p-4 rounded-lg border shadow-sm ${colorClasses[color] || colorClasses.gray}`}>
            <dt className="text-sm font-medium opacity-80 truncate">{label}</dt>
            <dd className="mt-1 text-3xl font-semibold">{value}</dd>
        </div>
    );
}

function ManifestViewer({ problem, configMaps }: { problem: any, configMaps: { name: string, data: any }[] }) {
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
