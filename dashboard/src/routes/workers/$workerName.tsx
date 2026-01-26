import { createFileRoute, Link, useRouter } from '@tanstack/react-router';
import { useState } from 'react';
import { getWorker, getProblemEnvironments, updateWorkerSchedule } from '../../data/k8s';
import yaml from 'js-yaml';
import { cleanManifest } from '../../utils/manifest';
import { ChevronLeft, Server, Activity, Box, FileCode, Cpu, HardDrive, ChevronDown, CalendarCheck, CalendarX, RefreshCw, LineChart } from 'lucide-react';
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
    const router = useRouter();
    const [isActionMenuOpen, setIsActionMenuOpen] = useState(false);
    const [activeModal, setActiveModal] = useState<'enable' | 'disable' | null>(null);
    const [confirmName, setConfirmName] = useState('');
    const [isProcessing, setIsProcessing] = useState(false);

    const handleAction = async (action: 'enable' | 'disable') => {
        if (confirmName !== worker.metadata.name) return;

        setIsProcessing(true);
        try {
            await updateWorkerSchedule({ data: { name: worker.metadata.name, disabled: action === 'disable' } });
            await router.invalidate();
            setActiveModal(null);
            setConfirmName('');
        } catch (err) {
            console.error(`Failed to ${action} schedule:`, err);
            alert(`Failed to ${action} worker schedule`);
        } finally {
            setIsProcessing(false);
        }
    };

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
                                    <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">Worker Class</dt>
                                    <dd className="mt-1 text-sm text-gray-900 dark:text-white">
                                        <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300">
                                            {worker.metadata.labels?.['netcon.janog.gr.jp/workerClass'] || '-'}
                                        </span>
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
            id: 'manifest',
            label: 'Manifest',
            icon: <FileCode className="w-4 h-4" />,
            content: (
                <Card title="Resource Manifest">
                    <pre className="p-4 bg-gray-900 text-gray-100 rounded-lg overflow-x-auto text-xs font-mono">
                        {yaml.dump(cleanManifest(worker))}
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
                    <Link to="/workers" search={{ q: '' }} className="inline-flex items-center text-sm text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 mb-4 transition-colors">
                        <ChevronLeft className="w-4 h-4 mr-1" />
                        Back to Workers
                    </Link>
                    <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4 w-full">
                        <div className="flex items-center space-x-3">
                            <div className="p-3 bg-gray-100 dark:bg-gray-800 rounded-lg">
                                <Server className="w-8 h-8 text-gray-700 dark:text-gray-300" />
                            </div>
                            <div>
                                <h1 className="text-3xl font-bold text-gray-900 dark:text-white">{worker.metadata.name}</h1>

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
                                                    href={`https://janog57-grafana.proelbtn.com/d/ad22s2f/telescope-3a-3a-worker?var-name=${worker.metadata.name}`}
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
                                                    search={{ worker: [worker.metadata.name] }}
                                                    className="flex items-center w-full px-4 py-2 text-sm text-left text-gray-700 dark:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700"
                                                    onClick={() => setIsActionMenuOpen(false)}
                                                >
                                                    <Activity className="mr-3 h-4 w-4" />
                                                    Search Environments
                                                </Link>
                                                <button
                                                    onClick={() => { setActiveModal('enable'); setIsActionMenuOpen(false); }}
                                                    disabled={!worker.spec.disableSchedule}
                                                    className={`flex items-center w-full px-4 py-2 text-sm text-left ${!worker.spec.disableSchedule
                                                        ? 'text-gray-400 dark:text-gray-600 cursor-not-allowed'
                                                        : 'text-green-600 dark:text-green-400 hover:bg-gray-100 dark:hover:bg-gray-700'
                                                        }`}
                                                >
                                                    <CalendarCheck className="mr-3 h-4 w-4" />
                                                    Enable Schedule
                                                </button>
                                                <button
                                                    onClick={() => { setActiveModal('disable'); setIsActionMenuOpen(false); }}
                                                    disabled={!!worker.spec.disableSchedule}
                                                    className={`flex items-center w-full px-4 py-2 text-sm text-left ${worker.spec.disableSchedule
                                                        ? 'text-gray-400 dark:text-gray-600 cursor-not-allowed'
                                                        : 'text-red-600 dark:text-red-400 hover:bg-gray-100 dark:hover:bg-gray-700'
                                                        }`}
                                                >
                                                    <CalendarX className="mr-3 h-4 w-4" />
                                                    Disable Schedule
                                                </button>
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

            {/* Modals */}
            {activeModal && (
                <div className="fixed inset-0 z-50 overflow-y-auto">
                    {/* Backdrop */}
                    <div
                        className="fixed inset-0 bg-black/60 backdrop-blur-sm transition-opacity"
                        aria-hidden="true"
                        onClick={() => setActiveModal(null)}
                    />

                    {/* Modal Centerer */}
                    <div className="flex min-h-full items-center justify-center p-4 text-center sm:p-0">
                        <div className="relative transform overflow-hidden rounded-lg bg-white dark:bg-gray-800 text-left shadow-xl transition-all sm:my-8 sm:w-full sm:max-w-lg">
                            <div className="bg-white dark:bg-gray-800 px-4 pt-5 pb-4 sm:p-6 sm:pb-4 border-b border-gray-100 dark:border-gray-700">
                                <div className="sm:flex sm:items-start">
                                    <div className={`mx-auto flex-shrink-0 flex items-center justify-center h-12 w-12 rounded-full sm:mx-0 sm:h-10 sm:w-10 ${activeModal === 'disable' ? 'bg-red-100 dark:bg-red-900/30' : 'bg-green-100 dark:bg-green-900/30'}`}>
                                        {activeModal === 'disable' ? (
                                            <CalendarX className="h-6 w-6 text-red-600 dark:text-red-400" />
                                        ) : (
                                            <CalendarCheck className="h-6 w-6 text-green-600 dark:text-green-400" />
                                        )}
                                    </div>
                                    <div className="mt-3 text-center sm:mt-0 sm:ml-4 sm:text-left w-full">
                                        <h3 className="text-lg leading-6 font-bold text-gray-900 dark:text-white">
                                            {activeModal === 'disable' ? 'Disable Schedule' : 'Enable Schedule'}
                                        </h3>
                                        <div className="mt-2">
                                            <p className="text-sm text-gray-500 dark:text-gray-400">
                                                {activeModal === 'disable'
                                                    ? 'This will disable scheduling for this worker. New environments will not be scheduled here. Are you sure?'
                                                    : 'This will enable scheduling for this worker. New environments may be scheduled here. Are you sure?'}
                                            </p>
                                        </div>
                                    </div>
                                </div>
                            </div>

                            <div className="px-4 py-5 sm:p-6 space-y-4">
                                <div>
                                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                        To confirm, type <span className="font-mono font-bold select-none text-gray-900 dark:text-white bg-gray-100 dark:bg-gray-700 px-1 rounded">{worker.metadata.name}</span> in the box below:
                                    </label>
                                    <input
                                        type="text"
                                        value={confirmName}
                                        onChange={(e) => setConfirmName(e.target.value)}
                                        className="block w-full rounded-md border-gray-300 dark:border-gray-600 dark:bg-gray-900 dark:text-white shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm px-3 py-2 cursor-text relative z-20"
                                        placeholder={worker.metadata.name}
                                        autoFocus
                                    />
                                </div>
                            </div>

                            <div className="bg-gray-50 dark:bg-gray-800/50 px-4 py-3 sm:px-6 sm:flex sm:flex-row-reverse gap-2">
                                <button
                                    type="button"
                                    disabled={confirmName !== worker.metadata.name || isProcessing}
                                    onClick={() => handleAction(activeModal)}
                                    className={`w-full inline-flex justify-center rounded-md border border-transparent px-4 py-2 text-base font-medium text-white shadow-sm sm:w-auto sm:text-sm ${activeModal === 'disable' ? 'bg-red-600 hover:bg-red-700 focus:ring-red-500' : 'bg-green-600 hover:bg-green-700 focus:ring-green-500'} ${(confirmName !== worker.metadata.name || isProcessing) ? 'opacity-50 cursor-not-allowed' : ''}`}
                                >
                                    {isProcessing && <RefreshCw className="animate-spin -ml-1 mr-2 h-4 w-4" />}
                                    Confirm {activeModal === 'disable' ? 'Disable' : 'Enable'}
                                </button>
                                <button
                                    type="button"
                                    onClick={() => { setActiveModal(null); setConfirmName(''); }}
                                    className="mt-3 w-full inline-flex justify-center rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-4 py-2 text-base font-medium text-gray-700 dark:text-gray-200 shadow-sm hover:bg-gray-50 dark:hover:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-indigo-500 sm:mt-0 sm:w-auto sm:text-sm"
                                >
                                    Cancel
                                </button>
                            </div>
                        </div>
                    </div>
                </div>
            )}
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
