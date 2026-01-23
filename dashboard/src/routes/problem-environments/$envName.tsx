import { createFileRoute, Link } from '@tanstack/react-router';
import { getProblemEnvironment } from '../../data/k8s';
import { ChevronLeft, Server, Activity, Terminal, Key } from 'lucide-react';

export const Route = createFileRoute('/problem-environments/$envName')({
    component: ProblemEnvironmentDetailPage,
    loader: async ({ params }) => {
        return await getProblemEnvironment({ data: params.envName });
    },
});

function ProblemEnvironmentDetailPage() {
    const env = Route.useLoaderData();

    return (
        <div className="p-6 bg-gray-50 dark:bg-gray-900 min-h-screen">
            <div className="max-w-7xl mx-auto space-y-6">
                {/* Header */}
                <div>
                    <Link to="/problem-environments" className="inline-flex items-center text-sm text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 mb-4 transition-colors">
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

                {/* Info Grid */}
                <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                    {/* General Info & Password */}
                    <div className="space-y-6">
                        <div className="bg-white dark:bg-gray-800 shadow rounded-lg p-6">
                            <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4 flex items-center">
                                <Activity className="w-5 h-5 mr-2" />
                                Status Conditions
                            </h2>
                            <div className="space-y-4">
                                {env.status?.conditions?.map((cond, idx) => (
                                    <div key={idx} className="flex items-start">
                                        <span className={`flex-shrink-0 w-3 h-3 mt-1.5 rounded-full mr-3 ${cond.status === 'True' ? 'bg-green-500' : 'bg-gray-300 dark:bg-gray-600'
                                            }`} />
                                        <div className="flex-1">
                                            <div className="flex items-center justify-between">
                                                <h3 className="text-sm font-medium text-gray-900 dark:text-white">{cond.type}</h3>
                                                <span className="text-xs text-gray-500 dark:text-gray-400">
                                                    {cond.lastTransitionTime ? new Date(cond.lastTransitionTime).toLocaleString() : '-'}
                                                </span>
                                            </div>
                                            <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
                                                Status: <span className="font-medium">{cond.status}</span>
                                                {cond.reason && <span className="ml-2 text-xs bg-gray-100 dark:bg-gray-700 px-2 py-0.5 rounded">Reason: {cond.reason}</span>}
                                            </p>
                                            {cond.message && <p className="text-xs text-gray-400 dark:text-gray-500 mt-1">{cond.message}</p>}
                                        </div>
                                    </div>
                                )) ?? <p className="text-sm text-gray-500">No conditions available.</p>}
                            </div>
                        </div>

                        <div className="bg-white dark:bg-gray-800 shadow rounded-lg p-6">
                            <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4 flex items-center">
                                <Key className="w-5 h-5 mr-2" />
                                Access Information
                            </h2>
                            <dl>
                                <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">Password</dt>
                                <dd className="mt-1 flex items-center">
                                    <code className="px-2 py-1 bg-gray-100 dark:bg-gray-900 rounded text-sm font-mono break-all text-gray-800 dark:text-gray-200">
                                        {env.status?.password || 'No password set'}
                                    </code>
                                </dd>
                            </dl>
                        </div>
                    </div>

                    {/* Containers */}
                    <div className="bg-white dark:bg-gray-800 shadow rounded-lg p-6">
                        <h2 className="text-lg font-semibold text-gray-900 dark:text-white mb-4 flex items-center">
                            <Terminal className="w-5 h-5 mr-2" />
                            Containers
                        </h2>
                        <div className="space-y-4">
                            {env.status?.containers?.map((container, idx) => (
                                <div key={idx} className="border border-gray-200 dark:border-gray-700 rounded-lg p-4">
                                    <div className="flex items-center justify-between mb-2">
                                        <h3 className="font-medium text-gray-900 dark:text-white">{container.name}</h3>
                                        <span className={`px-2 py-0.5 rounded text-xs font-medium ${container.ready
                                                ? 'bg-green-100 text-green-800 dark:bg-green-900/50 dark:text-green-300'
                                                : 'bg-red-100 text-red-800 dark:bg-red-900/50 dark:text-red-300'
                                            }`}>
                                            {container.ready ? 'Ready' : 'Not Ready'}
                                        </span>
                                    </div>
                                    <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 text-sm">
                                        <div>
                                            <span className="text-gray-500 dark:text-gray-400 block text-xs">Image</span>
                                            <span className="text-gray-900 dark:text-white break-all">{container.image}</span>
                                        </div>
                                        <div>
                                            <span className="text-gray-500 dark:text-gray-400 block text-xs">Container ID</span>
                                            <span className="text-gray-900 dark:text-white text-xs font-mono break-all">{container.containerID || '-'}</span>
                                        </div>
                                        <div className="sm:col-span-2">
                                            <span className="text-gray-500 dark:text-gray-400 block text-xs">Management IP</span>
                                            <span className="text-gray-900 dark:text-white font-mono">{container.managementIPAddress || '-'}</span>
                                        </div>
                                    </div>
                                </div>
                            )) ?? <p className="text-sm text-gray-500">No containers reported.</p>}
                        </div>
                    </div>
                </div>
            </div>
        </div>
    );
}
