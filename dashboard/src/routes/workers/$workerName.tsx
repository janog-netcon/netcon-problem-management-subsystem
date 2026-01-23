import { createFileRoute, Link } from '@tanstack/react-router';
import { getWorker, getProblemEnvironments } from '../../data/k8s';
import { ChevronLeft, Server, Activity, Box, FileCode, Cpu, HardDrive } from 'lucide-react';
import { Tabs } from '../../components/Tabs';
import { Card } from '../../components/Card';

export const Route = createFileRoute('/workers/$workerName')({
    component: WorkerDetailPage,
    loader: async ({ params }) => {
        const [worker, envsList] = await Promise.all([
            getWorker({ data: params.workerName }),
            getProblemEnvironments(),
        ]);
        return { worker, envsList };
    },
});

function WorkerDetailPage() {
    const { worker, envsList } = Route.useLoaderData();

    // Filter environments running on this worker
    const runningEnvs = envsList.items.filter(env => env.spec.workerName === worker.metadata.name);

    // Calculate resource usage
    const cpuUsage = parseFloat(worker.status?.workerInfo?.cpuUsedPercent || '0');
    const memUsage = parseFloat(worker.status?.workerInfo?.memoryUsedPercent || '0');

    const tabs = [
        {
            id: 'overview',
            label: 'Overview',
            icon: <Activity className="w-4 h-4" />,
            content: (
                <div className="space-y-6">
                    <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                        <StatusCard
                            label="CPU Usage"
                            value={`${cpuUsage.toFixed(1)}%`}
                            progress={cpuUsage}
                            icon={<Cpu className="w-5 h-5" />}
                            color={cpuUsage > 80 ? 'red' : 'blue'}
                        />
                        <StatusCard
                            label="Memory Usage"
                            value={`${memUsage.toFixed(1)}%`}
                            progress={memUsage}
                            icon={<HardDrive className="w-5 h-5" />}
                            color={memUsage > 80 ? 'red' : 'purple'}
                        />
                        <StatusCard
                            label="Running Environments"
                            value={runningEnvs.length}
                            icon={<Box className="w-5 h-5" />}
                            color="green"
                        />
                    </div>

                    <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                        <Card title="Node Information">
                            <dl className="grid grid-cols-1 gap-x-4 gap-y-6 sm:grid-cols-2">
                                <div className="sm:col-span-1">
                                    <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">Hostname</dt>
                                    <dd className="mt-1 text-sm text-gray-900 dark:text-white">{worker.status?.workerInfo?.hostname || '-'}</dd>
                                </div>
                                <div className="sm:col-span-1">
                                    <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">Address</dt>
                                    <dd className="mt-1 text-sm font-mono text-gray-900 dark:text-white">
                                        {worker.status?.workerInfo?.externalIPAddress}:{worker.status?.workerInfo?.externalPort}
                                    </dd>
                                </div>
                                <div className="sm:col-span-1">
                                    <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">Scheduling</dt>
                                    <dd className="mt-1 text-sm text-gray-900 dark:text-white">
                                        {worker.spec.disableSchedule ? (
                                            <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-300">
                                                Disabled
                                            </span>
                                        ) : (
                                            <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300">
                                                Enabled
                                            </span>
                                        )}
                                    </dd>
                                </div>
                                <div className="sm:col-span-1">
                                    <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">Status</dt>
                                    <dd className="mt-1 text-sm text-gray-900 dark:text-white">
                                        {worker.status?.conditions?.map(cond => (
                                            <div key={cond.type} className="flex items-center">
                                                <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${cond.status === 'True'
                                                    ? 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300'
                                                    : 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300'
                                                    }`}>
                                                    {cond.type}: {cond.status}
                                                </span>
                                            </div>
                                        ))}
                                    </dd>
                                </div>
                            </dl>
                        </Card>

                        <Card title="Conditions">
                            <div className="space-y-4">
                                {worker.status?.conditions?.map((cond, idx) => (
                                    <div key={idx} className="border-l-2 border-gray-200 dark:border-gray-700 pl-4 py-1">
                                        <div className="flex items-center justify-between">
                                            <span className="font-medium text-gray-900 dark:text-white">{cond.type}</span>
                                            <span className={`text-xs px-2 py-0.5 rounded ${cond.status === 'True' ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-600'}`}>
                                                {cond.status}
                                            </span>
                                        </div>
                                        <p className="text-sm text-gray-500 mt-1">{cond.message || cond.reason || 'No details available'}</p>
                                        <span className="text-xs text-gray-400 mt-1 block">
                                            Last Transition: {new Date(cond.lastTransitionTime).toLocaleString()}
                                        </span>
                                    </div>
                                ))}
                            </div>
                        </Card>
                    </div>
                </div>
            ),
        },
        {
            id: 'environments',
            label: 'Running Environments',
            icon: <Box className="w-4 h-4" />,
            content: (
                <Card title={`Environments on ${worker.metadata.name}`}>
                    <div className="overflow-x-auto">
                        <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
                            <thead className="bg-gray-50 dark:bg-gray-700">
                                <tr>
                                    <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Name</th>
                                    <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Status</th>
                                    <th scope="col" className="relative px-6 py-3"><span className="sr-only">View</span></th>
                                </tr>
                            </thead>
                            <tbody className="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
                                {runningEnvs.map(env => (
                                    <tr key={env.metadata.name} className="hover:bg-gray-50 dark:hover:bg-gray-700/50">
                                        <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900 dark:text-white">
                                            <Link to="/problem-environments/$envName" params={{ envName: env.metadata.name }} className="hover:underline text-indigo-600 dark:text-indigo-400">
                                                {env.metadata.name}
                                            </Link>
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
                                {runningEnvs.length === 0 && (
                                    <tr>
                                        <td colSpan={3} className="px-6 py-10 text-center text-sm text-gray-500 dark:text-gray-400">
                                            No environments currently running on this worker.
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
                        {JSON.stringify(worker, null, 2)}
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
                    <Link to="/workers" search={{ p: 1, q: '' }} className="inline-flex items-center text-sm text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 mb-4 transition-colors">
                        <ChevronLeft className="w-4 h-4 mr-1" />
                        Back to Workers
                    </Link>
                    <div className="flex items-center space-x-3">
                        <div className="p-3 bg-gray-100 dark:bg-gray-800 rounded-lg">
                            <Server className="w-8 h-8 text-gray-700 dark:text-gray-300" />
                        </div>
                        <div>
                            <h1 className="text-3xl font-bold text-gray-900 dark:text-white">{worker.metadata.name}</h1>
                            <p className="text-sm text-gray-500 dark:text-gray-400">Created on {new Date(worker.metadata.creationTimestamp).toLocaleDateString()}</p>
                        </div>
                    </div>
                </div>

                <Tabs tabs={tabs} />
            </div>
        </div>
    );
}

function StatusCard({ label, value, progress, icon, color }: { label: string, value: number | string, progress?: number, icon?: React.ReactNode, color: string }) {
    const colorClasses: Record<string, string> = {
        gray: 'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200 border-gray-200 dark:border-gray-700',
        blue: 'bg-blue-50 text-blue-800 dark:bg-blue-900/30 dark:text-blue-200 border-blue-200 dark:border-blue-900',
        green: 'bg-green-50 text-green-800 dark:bg-green-900/30 dark:text-green-200 border-green-200 dark:border-green-900',
        purple: 'bg-purple-50 text-purple-800 dark:bg-purple-900/30 dark:text-purple-200 border-purple-200 dark:border-purple-900',
        red: 'bg-red-50 text-red-800 dark:bg-red-900/30 dark:text-red-200 border-red-200 dark:border-red-900',
    };

    const progressColors: Record<string, string> = {
        blue: 'bg-blue-500',
        purple: 'bg-purple-500',
        green: 'bg-green-500',
        red: 'bg-red-500',
        gray: 'bg-gray-500',
    };

    return (
        <div className={`p-4 rounded-lg border shadow-sm ${colorClasses[color] || colorClasses.gray}`}>
            <div className="flex items-center justify-between mb-2">
                <dt className="text-sm font-medium opacity-80">{label}</dt>
                {icon && <div className="opacity-70">{icon}</div>}
            </div>
            <dd className="text-3xl font-semibold">{value}</dd>
            {progress !== undefined && (
                <div className="w-full bg-black/10 dark:bg-white/10 rounded-full h-2 mt-3">
                    <div
                        className={`h-2 rounded-full ${progressColors[color] || 'bg-gray-500'}`}
                        style={{ width: `${Math.min(100, progress)}%` }}
                    ></div>
                </div>
            )}
        </div>
    );
}
