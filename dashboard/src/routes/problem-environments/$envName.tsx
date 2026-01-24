import { createFileRoute, Link, useRouter } from '@tanstack/react-router';
import { useState } from 'react';
import { getProblemEnvironment, getDeploymentLog, assignProblemEnvironment, unassignProblemEnvironment, deleteProblemEnvironment, getWorker } from '../../data/k8s';
import { getStatusColor, getStatusText } from '../../data/status';
import { ChevronLeft, Server, Activity, Terminal, Key, CheckCircle, Clock, FileText, FileCode, ChevronDown, UserPlus, UserMinus, Trash2, AlertTriangle, RefreshCw, ShieldCheck, User } from 'lucide-react';
import { Card } from '../../components/Card';
import { CopyButton } from '../../components/CopyButton';
import { AnsiText } from '../../components/AnsiText';
import { Tabs } from '../../components/Tabs';

export const Route = createFileRoute('/problem-environments/$envName')({
    component: ProblemEnvironmentDetailPage,
    loader: async ({ params }) => {
        const [env, deployLog] = await Promise.all([
            getProblemEnvironment({ data: params.envName }),
            getDeploymentLog({ data: params.envName })
        ]);

        let worker = null;
        if (env.spec.workerName) {
            try {
                worker = await getWorker({ data: env.spec.workerName });
            } catch (err) {
                console.error(`Failed to fetch worker ${env.spec.workerName}:`, err);
            }
        }

        return { env, deployLog, worker };
    },
});

function ProblemEnvironmentDetailPage() {
    const { env, deployLog, worker } = Route.useLoaderData();
    const router = useRouter();
    const [isActionMenuOpen, setIsActionMenuOpen] = useState(false);
    const [activeModal, setActiveModal] = useState<'assign' | 'unassign' | 'delete' | null>(null);
    const [confirmName, setConfirmName] = useState('');
    const [isAssignedDeleteConfirmed, setIsAssignedDeleteConfirmed] = useState(false);
    const [isProcessing, setIsProcessing] = useState(false);

    const isReady = env.status?.conditions?.find(c => c.type === 'Ready' && c.status === 'True');
    const isAssigned = env.status?.conditions?.find(c => c.type === 'Assigned' && c.status === 'True');

    const handleAction = async (action: 'assign' | 'unassign' | 'delete') => {
        if (confirmName !== env.metadata.name) return;
        if (action === 'delete' && isAssigned && !isAssignedDeleteConfirmed) return;

        setIsProcessing(true);
        try {
            if (action === 'assign') {
                await assignProblemEnvironment({ data: env.metadata.name });
            } else if (action === 'unassign') {
                await unassignProblemEnvironment({ data: env.metadata.name });
            } else if (action === 'delete') {
                await deleteProblemEnvironment({ data: env.metadata.name });
                router.navigate({ to: '/problem-environments' });
                return;
            }
            await router.invalidate();
            setActiveModal(null);
            setConfirmName('');
            setIsAssignedDeleteConfirmed(false);
        } catch (err) {
            console.error(`Failed to perform ${action}:`, err);
            alert(`Failed to perform ${action}`);
        } finally {
            setIsProcessing(false);
        }
    };

    // Helper to extract SSH info from worker
    const workerIP = worker?.status?.workerInfo?.externalIPAddress;
    const workerPort = worker?.status?.workerInfo?.externalPort;
    const envName = env.metadata.name;

    const userSSHCommand = workerIP && workerPort
        ? `ssh nc_${envName}@${workerIP} -p ${workerPort}`
        : 'Worker info not available';

    const adminSSHCommand = workerIP && workerPort
        ? `ssh ncadmin_${envName}@${workerIP} -p ${workerPort}`
        : 'Worker info not available';

    const tabs = [
        {
            id: 'overview',
            label: 'Overview',
            icon: <Activity className="w-4 h-4" />,
            content: (
                <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
                    {/* Left Column: Connection & Status */}
                    <div className="lg:col-span-1 space-y-6">
                        <Card title={<><Key className="w-5 h-5 mr-2" /> Connection Info</>}>
                            <div className="space-y-4">
                                <div>
                                    <div className="flex items-center mb-1">
                                        <User className="w-4 h-4 mr-1 text-gray-500" />
                                        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">User SSH Command</label>
                                    </div>
                                    <div className="mt-1 flex items-center justify-between p-2 bg-gray-100 dark:bg-gray-900 rounded font-mono text-sm break-all text-gray-900 dark:text-white">
                                        <span>{userSSHCommand}</span>
                                        <CopyButton text={userSSHCommand} />
                                    </div>
                                </div>

                                <div>
                                    <div className="flex items-center mb-1">
                                        <ShieldCheck className="w-4 h-4 mr-1 text-indigo-500" />
                                        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">Admin SSH Command</label>
                                    </div>
                                    <div className="mt-1 flex items-center justify-between p-2 bg-gray-100 dark:bg-gray-900 rounded font-mono text-sm break-all text-gray-900 dark:text-white">
                                        <span>{adminSSHCommand}</span>
                                        <CopyButton text={adminSSHCommand} />
                                    </div>
                                </div>

                                <div>
                                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">Password</label>
                                    <div className="mt-1 flex items-center justify-between p-2 bg-gray-100 dark:bg-gray-900 rounded font-mono text-sm break-all text-gray-900 dark:text-white">
                                        <span>{env.status?.password || 'No password set'}</span>
                                        {env.status?.password && <CopyButton text={env.status.password} />}
                                    </div>
                                </div>
                            </div>
                        </Card>
                    </div>

                    <div className="lg:col-span-2">
                        <Card title={<><Activity className="w-5 h-5 mr-2" /> Timeline</>}>
                            <div className="relative border-l-2 border-gray-200 dark:border-gray-700 ml-3 space-y-6 pb-2">
                                {['Scheduled', 'Deployed', 'Ready', 'Assigned'].map((type) => {
                                    const cond = env.status?.conditions?.find(c => c.type === type);
                                    const isCompleted = cond?.status === 'True';

                                    let Icon = Clock;
                                    let colorClass = 'text-gray-400 bg-gray-100 dark:bg-gray-800';

                                    if (isCompleted) {
                                        Icon = CheckCircle;
                                        colorClass = 'text-green-500 bg-green-100 dark:bg-green-900/30';
                                    }

                                    return (
                                        <div key={type} className="relative pl-8">
                                            <span className={`absolute -left-2.5 top-0 flex items-center justify-center w-5 h-5 rounded-full ring-4 ring-white dark:ring-gray-800 ${colorClass}`}>
                                                <Icon className="w-3 h-3" />
                                            </span>
                                            <div className="flex flex-col">
                                                <span className={`text-sm font-medium ${isCompleted ? 'text-gray-900 dark:text-white' : 'text-gray-500 dark:text-gray-400'}`}>{type}</span>
                                                {cond?.lastTransitionTime && cond.status === 'True' && (
                                                    <span className="text-xs text-gray-400 dark:text-gray-500">{new Date(cond.lastTransitionTime).toLocaleString()}</span>
                                                )}
                                            </div>
                                        </div>
                                    );
                                })}
                            </div>
                        </Card>
                    </div>
                </div>
            )
        },
        {
            id: 'containers',
            label: 'Containers',
            icon: <Terminal className="w-4 h-4" />,
            content: (
                <Card title={<><Terminal className="w-5 h-5 mr-2" /> Containers</>}>
                    <div className="overflow-hidden">
                        <ul className="divide-y divide-gray-200 dark:divide-gray-700">
                            {env.status?.containers?.map((container, idx) => (
                                <li key={idx} className="py-4 first:pt-0 last:pb-0">
                                    <div className="flex items-center justify-between mb-2">
                                        <div className="flex items-center">
                                            <div className={`w-2.5 h-2.5 rounded-full mr-3 ${container.ready ? 'bg-green-500' : 'bg-red-500'}`}></div>
                                            <h3 className="text-sm font-medium text-gray-900 dark:text-white">{container.name}</h3>
                                        </div>
                                        <span className="text-xs text-gray-500 dark:text-gray-400 font-mono">{container.containerID ? container.containerID.slice(0, 12) : '-'}</span>
                                    </div>
                                    <div className="grid grid-cols-1 sm:grid-cols-2 gap-x-4 gap-y-2 text-sm text-gray-500 dark:text-gray-400 pl-5">
                                        <div className="flex flex-col">
                                            <span className="text-xs uppercase tracking-wider text-gray-400">Image</span>
                                            <span className="truncate" title={container.image}>{container.image}</span>
                                        </div>
                                        <div className="flex flex-col">
                                            <span className="text-xs uppercase tracking-wider text-gray-400">Mgmt IP</span>
                                            <span className="font-mono">{container.managementIPAddress || '-'}</span>
                                        </div>
                                    </div>
                                </li>
                            ))}
                            {(!env.status?.containers || env.status.containers.length === 0) && (
                                <li className="text-sm text-gray-500 dark:text-gray-400 italic">No containers reported yet.</li>
                            )}
                        </ul>
                    </div>
                </Card>
            )
        },
        {
            id: 'logs',
            label: 'Deployment Logs',
            icon: <FileText className="w-4 h-4" />,
            content: (
                <Card title={<><FileText className="w-5 h-5 mr-2" /> Deployment Logs</>}>
                    <LogViewer logs={deployLog} />
                </Card>
            )
        },
        {
            id: 'yaml',
            label: 'YAML',
            icon: <FileCode className="w-4 h-4" />,
            content: (
                <Card title="Raw Resource">
                    <pre className="p-4 bg-gray-900 text-gray-100 rounded-lg overflow-x-auto text-xs font-mono max-h-[600px]">
                        {JSON.stringify(env, null, 2)}
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
                    <Link to="/problem-environments" className="inline-flex items-center text-sm text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 mb-4 transition-colors">
                        <ChevronLeft className="w-4 h-4 mr-1" />
                        Back to Environments
                    </Link>
                    <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
                        <div className="flex items-center space-x-3">
                            <div className="p-3 bg-teal-100 dark:bg-teal-900/50 rounded-lg">
                                <Server className="w-8 h-8 text-teal-600 dark:text-teal-400" />
                            </div>
                            <div>
                                <div className="flex items-center space-x-2">
                                    <h1 className="text-3xl font-bold text-gray-900 dark:text-white">{env.metadata.name}</h1>
                                    <span className={`px-2.5 py-0.5 rounded-full text-xs font-semibold ${getStatusColor(env.status)}`}>
                                        {getStatusText(env.status)}
                                    </span>
                                </div>
                                <p className="text-sm text-gray-500 dark:text-gray-400">
                                    Deployed on <span className="font-medium text-gray-900 dark:text-gray-300">{env.spec.workerName || 'Pending'}</span>
                                </p>
                            </div>
                        </div>

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
                                            {env.metadata.ownerReferences?.find(ref => ref.kind === 'Problem') && (
                                                <Link
                                                    to="/problems/$problemName"
                                                    params={{ problemName: env.metadata.ownerReferences.find(ref => ref.kind === 'Problem')!.name }}
                                                    className="flex items-center w-full px-4 py-2 text-sm text-left text-gray-700 dark:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700"
                                                >
                                                    <Activity className="mr-3 h-4 w-4" />
                                                    See Problem
                                                </Link>
                                            )}
                                            {env.spec.workerName && (
                                                <Link
                                                    to="/workers/$workerName"
                                                    params={{ workerName: env.spec.workerName }}
                                                    className="flex items-center w-full px-4 py-2 text-sm text-left text-gray-700 dark:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700"
                                                >
                                                    <Server className="mr-3 h-4 w-4" />
                                                    See Worker
                                                </Link>
                                            )}
                                            <div className="border-t border-gray-100 dark:border-gray-700 my-1"></div>
                                            <button
                                                disabled={!isReady || !!isAssigned}
                                                onClick={() => { setActiveModal('assign'); setIsActionMenuOpen(false); }}
                                                className={`flex items-center w-full px-4 py-2 text-sm text-left ${(!isReady || !!isAssigned) ? 'text-gray-400 dark:text-gray-600 cursor-not-allowed' : 'text-gray-700 dark:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700'}`}
                                            >
                                                <UserPlus className="mr-3 h-4 w-4" />
                                                Assign Environment
                                            </button>
                                            <button
                                                disabled={!isAssigned}
                                                onClick={() => { setActiveModal('unassign'); setIsActionMenuOpen(false); }}
                                                className={`flex items-center w-full px-4 py-2 text-sm text-left ${!isAssigned ? 'text-gray-400 dark:text-gray-600 cursor-not-allowed' : 'text-gray-700 dark:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700'}`}
                                            >
                                                <UserMinus className="mr-3 h-4 w-4" />
                                                Unassign Environment
                                            </button>
                                            <div className="border-t border-gray-100 dark:border-gray-700 my-1"></div>
                                            <button
                                                onClick={() => { setActiveModal('delete'); setIsActionMenuOpen(false); }}
                                                className="flex items-center w-full px-4 py-2 text-sm text-left text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20"
                                            >
                                                <Trash2 className="mr-3 h-4 w-4" />
                                                Delete Environment
                                            </button>
                                        </div>
                                    </div>
                                </>
                            )}
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
                                    <div className={`mx-auto flex-shrink-0 flex items-center justify-center h-12 w-12 rounded-full sm:mx-0 sm:h-10 sm:w-10 ${activeModal === 'delete' ? 'bg-red-100 dark:bg-red-900/30' : 'bg-indigo-100 dark:bg-indigo-900/30'}`}>
                                        {activeModal === 'delete' ? (
                                            <Trash2 className="h-6 w-6 text-red-600 dark:text-red-400" />
                                        ) : activeModal === 'assign' ? (
                                            <UserPlus className="h-6 w-6 text-indigo-600 dark:text-indigo-400" />
                                        ) : (
                                            <UserMinus className="h-6 w-6 text-indigo-600 dark:text-indigo-400" />
                                        )}
                                    </div>
                                    <div className="mt-3 text-center sm:mt-0 sm:ml-4 sm:text-left w-full">
                                        <h3 className="text-lg leading-6 font-bold text-gray-900 dark:text-white">
                                            {activeModal === 'assign' ? 'Assign Environment' : activeModal === 'unassign' ? 'Unassign Environment' : 'Delete Environment'}
                                        </h3>
                                        <div className="mt-2">
                                            <p className="text-sm text-gray-500 dark:text-gray-400">
                                                {activeModal === 'assign'
                                                    ? 'This will mark the environment as assigned. Are you sure?'
                                                    : activeModal === 'unassign'
                                                        ? 'This will unassign the environment. Are you sure?'
                                                        : 'This action cannot be undone. This will permanently delete the environment configuration and all associated resources.'}
                                            </p>
                                        </div>
                                    </div>
                                </div>
                            </div>

                            <div className="px-4 py-5 sm:p-6 space-y-4">
                                <div>
                                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                                        To confirm, type <span className="font-mono font-bold select-none text-gray-900 dark:text-white bg-gray-100 dark:bg-gray-700 px-1 rounded">{env.metadata.name}</span> in the box below:
                                    </label>
                                    <input
                                        type="text"
                                        value={confirmName}
                                        onChange={(e) => setConfirmName(e.target.value)}
                                        className="block w-full rounded-md border-gray-300 dark:border-gray-600 dark:bg-gray-900 dark:text-white shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm px-3 py-2 cursor-text relative z-20"
                                        placeholder={env.metadata.name}
                                        autoFocus
                                    />
                                </div>

                                {activeModal === 'delete' && isAssigned && (
                                    <div className="p-3 bg-red-50 dark:bg-red-900/20 border border-red-100 dark:border-red-900/30 rounded-md">
                                        <div className="flex">
                                            <div className="flex-shrink-0">
                                                <AlertTriangle className="h-5 w-5 text-red-400" aria-hidden="true" />
                                            </div>
                                            <div className="ml-3">
                                                <h3 className="text-sm font-medium text-red-800 dark:text-red-300">Warning: Environment is Assigned</h3>
                                                <div className="mt-2">
                                                    <div className="flex items-center">
                                                        <input
                                                            id="confirm-assigned-delete"
                                                            type="checkbox"
                                                            checked={isAssignedDeleteConfirmed}
                                                            onChange={(e) => setIsAssignedDeleteConfirmed(e.target.checked)}
                                                            className="h-4 w-4 text-red-600 focus:ring-red-500 border-gray-300 rounded cursor-pointer"
                                                        />
                                                        <label htmlFor="confirm-assigned-delete" className="ml-2 block text-sm text-red-700 dark:text-red-400 cursor-pointer">
                                                            I am sure I want to delete an ASSIGNED environment
                                                        </label>
                                                    </div>
                                                </div>
                                            </div>
                                        </div>
                                    </div>
                                )}
                            </div>

                            <div className="bg-gray-50 dark:bg-gray-800/50 px-4 py-3 sm:px-6 sm:flex sm:flex-row-reverse gap-2">
                                <button
                                    type="button"
                                    disabled={confirmName !== env.metadata.name || (activeModal === 'delete' && isAssigned && !isAssignedDeleteConfirmed) || isProcessing}
                                    onClick={() => handleAction(activeModal)}
                                    className={`w-full inline-flex justify-center rounded-md border border-transparent px-4 py-2 text-base font-medium text-white shadow-sm sm:w-auto sm:text-sm ${activeModal === 'delete' ? 'bg-red-600 hover:bg-red-700 focus:ring-red-500' : 'bg-indigo-600 hover:bg-indigo-700 focus:ring-indigo-500'} ${(confirmName !== env.metadata.name || (activeModal === 'delete' && isAssigned && !isAssignedDeleteConfirmed) || isProcessing) ? 'opacity-50 cursor-not-allowed' : ''}`}
                                >
                                    {isProcessing && <RefreshCw className="animate-spin -ml-1 mr-2 h-4 w-4" />}
                                    Confirm {activeModal === 'delete' ? 'Delete' : activeModal === 'assign' ? 'Assign' : 'Unassign'}
                                </button>
                                <button
                                    type="button"
                                    onClick={() => { setActiveModal(null); setConfirmName(''); setIsAssignedDeleteConfirmed(false); }}
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

function LogViewer({ logs }: { logs: { stdout: string | null, stderr: string | null } | null }) {
    const [activeTab, setActiveTab] = useState<'stderr' | 'stdout'>('stdout');

    if (!logs || (!logs.stdout && !logs.stderr)) {
        return <div className="p-4 text-sm text-gray-500 italic dark:text-gray-400">No deployment logs found.</div>;
    }

    return (
        <div className="bg-gray-900 rounded-lg overflow-hidden font-mono text-xs">
            {/* Tabs */}
            <div className="flex border-b border-gray-700">
                <button
                    onClick={() => setActiveTab('stdout')}
                    className={`px-4 py-2 font-medium focus:outline-none transition-colors ${activeTab === 'stdout'
                        ? 'bg-gray-800 text-white border-b-2 border-indigo-500'
                        : 'text-gray-400 hover:text-gray-200 hover:bg-gray-800/50'
                        }`}
                >
                    stdout
                </button>
                <button
                    onClick={() => setActiveTab('stderr')}
                    className={`px-4 py-2 font-medium focus:outline-none transition-colors ${activeTab === 'stderr'
                        ? 'bg-gray-800 text-white border-b-2 border-indigo-500'
                        : 'text-gray-400 hover:text-gray-200 hover:bg-gray-800/50'
                        }`}
                >
                    stderr
                </button>
            </div>

            {/* Content */}
            <div className="max-h-96 overflow-auto p-4">
                <pre className="text-gray-300 whitespace-pre font-mono leading-relaxed" style={{ fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace' }}>
                    <AnsiText text={activeTab === 'stderr' ? (logs.stderr || 'No stderr content.') : (logs.stdout || 'No stdout content.')} />
                </pre>
            </div>
        </div>
    );
}
