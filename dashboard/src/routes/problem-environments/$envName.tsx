import { createFileRoute, Link } from '@tanstack/react-router';
import { getProblemEnvironment } from '../../data/k8s';
import { ChevronLeft, Server, Activity, Terminal, Key, PlayCircle, CheckCircle, Clock } from 'lucide-react';
import { Card } from '../../components/Card';
import { CopyButton } from '../../components/CopyButton';

export const Route = createFileRoute('/problem-environments/$envName')({
    component: ProblemEnvironmentDetailPage,
    loader: async ({ params }) => {
        return await getProblemEnvironment({ data: params.envName });
    },
});

function ProblemEnvironmentDetailPage() {
    const env = Route.useLoaderData();

    // Helper to extract management IP
    const managementIP = env.status?.containers?.find(c => c.managementIPAddress)?.managementIPAddress;
    const sshCommand = managementIP ? `ssh user@${managementIP}` : 'IP not available';

    return (
        <div className="p-6 bg-gray-50 dark:bg-gray-900 min-h-screen">
            <div className="max-w-7xl mx-auto space-y-6">
                {/* Header */}
                <div>
                    <Link to="/problem-environments" search={{ p: 1, q: '' }} className="inline-flex items-center text-sm text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 mb-4 transition-colors">
                        <ChevronLeft className="w-4 h-4 mr-1" />
                        Back to Environments
                    </Link>
                    <div className="flex items-center space-x-3">
                        <div className="p-3 bg-teal-100 dark:bg-teal-900/50 rounded-lg">
                            <Server className="w-8 h-8 text-teal-600 dark:text-teal-400" />
                        </div>
                        <div>
                            <h1 className="text-3xl font-bold text-gray-900 dark:text-white">{env.metadata.name}</h1>
                            <p className="text-sm text-gray-500 dark:text-gray-400">
                                Deployed on <span className="font-medium text-gray-900 dark:text-gray-300">{env.spec.workerName || 'Pending'}</span>
                            </p>
                        </div>
                    </div>
                </div>

                <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
                    {/* Left Column: Connection & Status */}
                    <div className="lg:col-span-1 space-y-6">
                        <Card title={<><Key className="w-5 h-5 mr-2" /> Connection Info</>}>
                            <div className="space-y-4">
                                <div>
                                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">SSH Command</label>
                                    <div className="mt-1 flex rounded-md shadow-sm">
                                        <div className="relative flex-grow focus-within:z-10">
                                            <input type="text" readOnly value={sshCommand} className="focus:ring-indigo-500 focus:border-indigo-500 block w-full rounded-none rounded-l-md sm:text-sm border-gray-300 dark:border-gray-600 dark:bg-gray-700 dark:text-white px-3 py-2 font-mono" />
                                        </div>
                                        <div className="-ml-px relative inline-flex items-center space-x-2 border border-gray-300 dark:border-gray-600 dark:bg-gray-700 text-sm font-medium rounded-r-md text-gray-700 bg-gray-50 hover:bg-gray-100 focus:outline-none focus:ring-1 focus:ring-indigo-500 focus:border-indigo-500">
                                            <CopyButton text={sshCommand} className="border-0 bg-transparent shadow-none" />
                                        </div>
                                    </div>
                                </div>

                                <div>
                                    <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">Password</label>
                                    <div className="mt-1 flex items-center justify-between p-2 bg-gray-100 dark:bg-gray-900 rounded font-mono text-sm break-all">
                                        <span>{env.status?.password || 'No password set'}</span>
                                        {env.status?.password && <CopyButton text={env.status.password} />}
                                    </div>
                                </div>
                            </div>
                        </Card>

                        <Card title={<><Activity className="w-5 h-5 mr-2" /> Timeline</>}>
                            <div className="relative border-l-2 border-gray-200 dark:border-gray-700 ml-3 space-y-6 pb-2">
                                {['Scheduled', 'Deployed', 'Ready'].map((type, idx) => {
                                    const cond = env.status?.conditions?.find(c => c.type === type);
                                    const isCompleted = cond?.status === 'True';
                                    const isCurrent = !isCompleted && (idx === 0 || env.status?.conditions?.some(c => c.type === ['Scheduled', 'Deployed', 'Ready'][idx - 1] && c.status === 'True')); // Rough logic

                                    let Icon = Clock;
                                    let colorClass = 'text-gray-400 bg-gray-100 dark:bg-gray-800';

                                    if (isCompleted) {
                                        Icon = CheckCircle;
                                        colorClass = 'text-green-500 bg-green-100 dark:bg-green-900/30';
                                    } else if (isCurrent) {
                                        Icon = PlayCircle;
                                        colorClass = 'text-blue-500 bg-blue-100 dark:bg-blue-900/30';
                                    }

                                    return (
                                        <div key={type} className="relative pl-8">
                                            <span className={`absolute -left-2.5 top-0 flex items-center justify-center w-5 h-5 rounded-full ring-4 ring-white dark:ring-gray-800 ${colorClass}`}>
                                                <Icon className="w-3 h-3" />
                                            </span>
                                            <div className="flex flex-col">
                                                <span className={`text-sm font-medium ${isCompleted ? 'text-gray-900 dark:text-white' : 'text-gray-500 dark:text-gray-400'}`}>{type}</span>
                                                {cond?.lastTransitionTime && (
                                                    <span className="text-xs text-gray-400 dark:text-gray-500">{new Date(cond.lastTransitionTime).toLocaleString()}</span>
                                                )}
                                            </div>
                                        </div>
                                    );
                                })}
                            </div>
                        </Card>
                    </div>

                    {/* Right Column: Containers & Details */}
                    <div className="lg:col-span-2 space-y-6">
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

                        <Card title="Raw Status">
                            <pre className="p-4 bg-gray-900 text-gray-100 rounded-lg overflow-x-auto text-xs font-mono max-h-96">
                                {JSON.stringify(env.status, null, 2)}
                            </pre>
                        </Card>
                    </div>
                </div>
            </div>
        </div>
    );
}
