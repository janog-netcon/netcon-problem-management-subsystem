import { createFileRoute, useNavigate, useRouter } from '@tanstack/react-router';
import { getWorkers } from '../../data/k8s';
import { SearchBar } from '../../components/SearchBar';
import { z } from 'zod';
import { Cpu, HardDrive, AlertCircle, CheckCircle, RefreshCw } from 'lucide-react';

const workerSearchSchema = z.object({
    q: z.string().optional(),
});

export const Route = createFileRoute('/workers/')({
    component: WorkersPage,
    loader: async () => {
        const workers = await getWorkers();
        return { workers, updatedAt: new Date() };
    },
    validateSearch: (search) => workerSearchSchema.parse(search),
    staleTime: 60000,
    gcTime: 300000,
    shouldReload: false,
});

function WorkersPage() {
    const { workers, updatedAt } = Route.useLoaderData();
    const search = Route.useSearch();
    const navigate = useNavigate({ from: Route.fullPath });
    const router = useRouter();


    // Client-side filtering
    const filteredWorkers = workers.items.filter((worker: any) => {
        if (!search.q) return true;
        return worker.metadata.name.toLowerCase().includes(search.q.toLowerCase());
    });



    const handleSearch = (newQuery: string) => {
        navigate({
            search: (prev) => ({ ...prev, q: newQuery || undefined }),
            replace: true,
        });
    };



    return (
        <div className="p-6 bg-gray-50 dark:bg-gray-900 min-h-screen">
            <div className="max-w-7xl mx-auto">
                <div className="flex flex-col md:flex-row md:items-center md:justify-between mb-8">
                    <div>
                        <h1 className="text-3xl font-bold text-gray-900 dark:text-white">Workers</h1>
                        <p className="mt-2 text-sm text-gray-600 dark:text-gray-400">
                            Monitor infrastructure nodes and their resource usage.
                        </p>
                    </div>
                    <div className="mt-4 md:mt-0 flex flex-col items-end space-y-2">
                        <div className="flex items-center space-x-3 w-full md:w-auto">
                            <div className="w-full md:w-64">
                                <SearchBar value={search.q || ''} onChange={handleSearch} placeholder="Search workers..." />
                            </div>
                            <button
                                onClick={() => router.invalidate()}
                                className="p-2 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors focus:outline-none focus:ring-2 focus:ring-indigo-500"
                                title="Refresh data"
                            >
                                <RefreshCw className={`w-5 h-5 text-gray-500 dark:text-gray-400 ${router.state.isLoading ? 'animate-spin' : ''}`} />
                            </button>
                        </div>
                        <p className="text-[10px] text-gray-400 dark:text-gray-500 font-mono">
                            Last updated: {new Date(updatedAt).toLocaleTimeString()}
                        </p>
                    </div>
                </div>

                <div className="bg-white dark:bg-gray-800 shadow rounded-lg overflow-hidden">
                    <div className="overflow-x-auto">
                        <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
                            <thead className="bg-gray-50 dark:bg-gray-700">
                                <tr>
                                    <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                                        Name
                                    </th>
                                    <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                                        Class
                                    </th>
                                    <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                                        Status
                                    </th>
                                    <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                                        CPU Usage
                                    </th>
                                    <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                                        Memory Usage
                                    </th>
                                </tr>
                            </thead>
                            <tbody className="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
                                {filteredWorkers.map((worker: any) => {
                                    const isReady = worker.status?.conditions?.some((c: any) => c.type === 'Ready' && c.status === 'True');
                                    const cpuUsage = parseFloat(worker.status?.workerInfo?.cpuUsedPercent || '0');
                                    const memUsage = parseFloat(worker.status?.workerInfo?.memoryUsedPercent || '0');

                                    return (
                                        <tr
                                            key={worker.metadata.name}
                                            onClick={() => navigate({ to: '/workers/$workerName', params: { workerName: worker.metadata.name } })}
                                            className="hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors cursor-pointer"
                                        >
                                            <td className="px-6 py-4 whitespace-nowrap">
                                                <div className="text-sm font-medium text-gray-900 dark:text-white">
                                                    {worker.metadata.name}
                                                </div>
                                            </td>
                                            <td className="px-6 py-4 whitespace-nowrap">
                                                <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300">
                                                    {worker.metadata.labels?.['netcon.janog.gr.jp/workerClass'] || '-'}
                                                </span>
                                            </td>
                                            <td className="px-6 py-4 whitespace-nowrap">
                                                <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${isReady
                                                    ? 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300'
                                                    : 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-300'
                                                    }`}>
                                                    {isReady ? <CheckCircle className="w-3 h-3 mr-1" /> : <AlertCircle className="w-3 h-3 mr-1" />}
                                                    {isReady ? 'Ready' : 'Not Ready'}
                                                </span>
                                                {worker.spec.disableSchedule && (
                                                    <span className="ml-2 inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300">
                                                        Disabled
                                                    </span>
                                                )}
                                            </td>
                                            <td className="px-6 py-4 whitespace-nowrap">
                                                <div className="w-full max-w-[100px]">
                                                    <div className="flex justify-between text-xs mb-1">
                                                        <span className="text-gray-500 dark:text-gray-400"><Cpu className="w-3 h-3 inline mr-1" /></span>
                                                        <span className={`font-medium ${cpuUsage > 80 ? 'text-red-500' : 'text-gray-700 dark:text-gray-300'}`}>
                                                            {cpuUsage.toFixed(1)}%
                                                        </span>
                                                    </div>
                                                    <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-1.5">
                                                        <div
                                                            className={`h-1.5 rounded-full ${cpuUsage > 80 ? 'bg-red-500' : 'bg-blue-500'}`}
                                                            style={{ width: `${Math.min(100, cpuUsage)}%` }}
                                                        ></div>
                                                    </div>
                                                </div>
                                            </td>
                                            <td className="px-6 py-4 whitespace-nowrap">
                                                <div className="w-full max-w-[100px]">
                                                    <div className="flex justify-between text-xs mb-1">
                                                        <span className="text-gray-500 dark:text-gray-400"><HardDrive className="w-3 h-3 inline mr-1" /></span>
                                                        <span className={`font-medium ${memUsage > 80 ? 'text-red-500' : 'text-gray-700 dark:text-gray-300'}`}>
                                                            {memUsage.toFixed(1)}%
                                                        </span>
                                                    </div>
                                                    <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-1.5">
                                                        <div
                                                            className={`h-1.5 rounded-full ${memUsage > 80 ? 'bg-red-500' : 'bg-purple-500'}`}
                                                            style={{ width: `${Math.min(100, memUsage)}%` }}
                                                        ></div>
                                                    </div>
                                                </div>
                                            </td>
                                        </tr>
                                    );
                                })}
                                {filteredWorkers.length === 0 && (
                                    <tr>
                                        <td colSpan={5} className="px-6 py-10 text-center text-sm text-gray-500 dark:text-gray-400">
                                            No workers found.
                                        </td>
                                    </tr>
                                )}
                            </tbody>
                        </table>
                    </div>
                </div>
            </div>
        </div>
    );
}
