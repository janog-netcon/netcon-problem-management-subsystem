import { createFileRoute, Link } from '@tanstack/react-router';
import { getProblem, getProblemEnvironments, getWorkers } from '../../data/k8s';
import { ChevronLeft, Box, Activity, Network, FileCode } from 'lucide-react';
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
        return { problem, envsList, workers };
    },
});

function ProblemDetailPage() {
    const { problem, envsList, workers } = Route.useLoaderData();

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

                    <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                        <Card title="Specification">
                            <dl className="grid grid-cols-1 gap-x-4 gap-y-6 sm:grid-cols-2">
                                <div className="sm:col-span-1">
                                    <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">Assignable Replicas</dt>
                                    <dd className="mt-1 text-sm text-gray-900 dark:text-white">{problem.spec.assignableReplicas}</dd>
                                </div>
                                <div className="sm:col-span-1">
                                    <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">UID</dt>
                                    <dd className="mt-1 text-xs font-mono text-gray-900 dark:text-white">{problem.metadata.uid}</dd>
                                </div>
                            </dl>
                        </Card>
                        <Card title="Internal Status">
                            <div className="space-y-2">
                                <div className="flex justify-between">
                                    <span className="text-sm text-gray-500 dark:text-gray-400">Generation</span>
                                    <span className="text-sm font-mono text-gray-900 dark:text-white">{problem.metadata.generation ?? '-'}</span>
                                </div>
                                <div className="flex justify-between">
                                    <span className="text-sm text-gray-500 dark:text-gray-400">Resource Version</span>
                                    <span className="text-sm font-mono text-gray-900 dark:text-white">{problem.metadata.resourceVersion}</span>
                                </div>
                            </div>
                        </Card>
                    </div>

                    <Card title="Worker Distribution">
                        <div className="overflow-x-auto mt-2">
                            <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
                                <thead className="bg-gray-50 dark:bg-gray-700">
                                    <tr>
                                        <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Worker</th>
                                        <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Environment Count</th>
                                        <th scope="col" className="relative px-6 py-3"><span className="sr-only">View</span></th>
                                    </tr>
                                </thead>
                                <tbody className="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
                                    {workers.items.map(worker => {
                                        const count = relatedEnvs.filter(e => e.spec.workerName === worker.metadata.name).length;
                                        if (count === 0) return null;

                                        return (
                                            <tr key={worker.metadata.name} className="hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors">
                                                <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900 dark:text-white">
                                                    <div className="flex items-center">
                                                        <Network className="w-4 h-4 mr-2 text-gray-400" />
                                                        {worker.metadata.name}
                                                    </div>
                                                </td>
                                                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                                                    {count}
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
                                            <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
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
            id: 'environments',
            label: 'Environments',
            icon: <Box className="w-4 h-4" />,
            content: (
                <Card>
                    <div className="overflow-x-auto">
                        <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
                            <thead className="bg-gray-50 dark:bg-gray-700">
                                <tr>
                                    <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Name</th>
                                    <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Worker</th>
                                    <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Conditions</th>
                                    <th scope="col" className="relative px-6 py-3"><span className="sr-only">View</span></th>
                                </tr>
                            </thead>
                            <tbody className="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
                                {relatedEnvs.map(env => (
                                    <tr key={env.metadata.name} className="hover:bg-gray-50 dark:hover:bg-gray-700/50">
                                        <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900 dark:text-white">
                                            <Link to="/problem-environments/$envName" params={{ envName: env.metadata.name }} className="hover:underline text-indigo-600 dark:text-indigo-400">
                                                {env.metadata.name}
                                            </Link>
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                                            {env.spec.workerName || '-'}
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400 space-x-1">
                                            {env.status?.conditions?.map(cond => (
                                                <span key={cond.type} className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${cond.status === 'True'
                                                    ? 'bg-green-100 text-green-800 dark:bg-green-900/50 dark:text-green-300'
                                                    : 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300'
                                                    }`}>
                                                    {cond.type}
                                                </span>
                                            ))}
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                                            <Link to="/problem-environments/$envName" params={{ envName: env.metadata.name }} className="text-indigo-600 hover:text-indigo-900 dark:text-indigo-400 dark:hover:text-indigo-300">
                                                Details
                                            </Link>
                                        </td>
                                    </tr>
                                ))}
                                {relatedEnvs.length === 0 && (
                                    <tr>
                                        <td colSpan={4} className="px-6 py-10 text-center text-sm text-gray-500 dark:text-gray-400">
                                            No environments found.
                                        </td>
                                    </tr>
                                )}
                            </tbody>
                        </table>
                    </div>
                </Card>
            ),
        },
        {
            id: 'yaml',
            label: 'YAML',
            icon: <FileCode className="w-4 h-4" />,
            content: (
                <Card title="Raw Resource">
                    <pre className="p-4 bg-gray-900 text-gray-100 rounded-lg overflow-x-auto text-xs font-mono">
                        {JSON.stringify(problem, null, 2)}
                    </pre>
                </Card>
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
                    <div className="flex items-center space-x-3">
                        <div className="p-3 bg-indigo-100 dark:bg-indigo-900/50 rounded-lg">
                            <Box className="w-8 h-8 text-indigo-600 dark:text-indigo-400" />
                        </div>
                        <div>
                            <h1 className="text-3xl font-bold text-gray-900 dark:text-white">{problem.metadata.name}</h1>
                            <p className="text-sm text-gray-500 dark:text-gray-400">Created on {new Date(problem.metadata.creationTimestamp).toLocaleDateString()}</p>
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
