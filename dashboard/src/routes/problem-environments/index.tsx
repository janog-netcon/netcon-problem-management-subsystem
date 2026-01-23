import { createFileRoute, Link, useNavigate } from '@tanstack/react-router';
import { getProblemEnvironments, ProblemEnvironment } from '../../data/k8s';
import { SearchBar } from '../../components/SearchBar';
import { Pagination } from '../../components/Pagination';
import { z } from 'zod';

const envSearchSchema = z.object({
    p: z.number().default(1),
    q: z.string().default(''),
});

export const Route = createFileRoute('/problem-environments/')({
    component: ProblemEnvironmentsPage,
    loader: async () => await getProblemEnvironments(),
    validateSearch: (search) => envSearchSchema.parse(search),
});

function ProblemEnvironmentsPage() {
    const envsList = Route.useLoaderData();
    const search = Route.useSearch();
    const navigate = useNavigate({ from: Route.fullPath });

    const itemsPerPage = 20;

    const filteredEnvs = envsList.items.filter((env) => {
        if (!search.q) return true;
        return env.metadata.name.toLowerCase().includes(search.q.toLowerCase());
    });

    const totalItems = filteredEnvs.length;
    const totalPages = Math.ceil(totalItems / itemsPerPage);
    const currentPage = Math.min(Math.max(1, search.p), Math.max(1, totalPages));

    const paginatedEnvs = filteredEnvs.slice(
        (currentPage - 1) * itemsPerPage,
        currentPage * itemsPerPage
    );

    const handleSearch = (newQuery: string) => {
        navigate({
            search: (prev) => ({ ...prev, q: newQuery, p: 1 }),
            replace: true,
        });
    };

    const handlePageChange = (newPage: number) => {
        navigate({
            search: (prev) => ({ ...prev, p: newPage }),
        });
    };

    // Helper to determine status color based on conditions
    const getStatusColor = (status: ProblemEnvironment['status']) => {
        const isReady = status?.conditions?.find(c => c.type === 'Ready' && c.status === 'True');
        if (isReady) return 'bg-green-100 text-green-800 dark:bg-green-900/50 dark:text-green-300';

        const isDeployed = status?.conditions?.find(c => c.type === 'Deployed' && c.status === 'True');
        if (isDeployed) return 'bg-blue-100 text-blue-800 dark:bg-blue-900/50 dark:text-blue-300';

        const isScheduled = status?.conditions?.find(c => c.type === 'Scheduled' && c.status === 'True');
        if (isScheduled) return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/50 dark:text-yellow-300';

        return 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300';
    };

    const getStatusText = (status: ProblemEnvironment['status']) => {
        const isReady = status?.conditions?.find(c => c.type === 'Ready' && c.status === 'True');
        if (isReady) return 'Ready';

        const isDeployed = status?.conditions?.find(c => c.type === 'Deployed' && c.status === 'True');
        if (isDeployed) return 'Deployed';

        const isScheduled = status?.conditions?.find(c => c.type === 'Scheduled' && c.status === 'True');
        if (isScheduled) return 'Scheduled';

        return 'Unknown';
    };

    return (
        <div className="p-6 bg-gray-50 dark:bg-gray-900 min-h-screen">
            <div className="max-w-7xl mx-auto">
                <div className="flex flex-col md:flex-row md:items-center md:justify-between mb-8">
                    <div>
                        <h1 className="text-3xl font-bold text-gray-900 dark:text-white">Problem Environments</h1>
                        <p className="mt-2 text-sm text-gray-600 dark:text-gray-400">View and manage problem environments running on workers.</p>
                    </div>
                    <div className="mt-4 md:mt-0 w-full md:w-64">
                        <SearchBar value={search.q} onChange={handleSearch} placeholder="Search environments..." />
                    </div>
                </div>

                <div className="bg-white dark:bg-gray-800 shadow rounded-lg overflow-hidden">
                    <div className="overflow-x-auto">
                        <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
                            <thead className="bg-gray-50 dark:bg-gray-700">
                                <tr>
                                    <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Name</th>
                                    <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Worker</th>
                                    <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Main Status</th>
                                    <th scope="col" className="relative px-6 py-3"><span className="sr-only">View</span></th>
                                </tr>
                            </thead>
                            <tbody className="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
                                {paginatedEnvs.map((env) => (
                                    <tr key={env.metadata.name} className="hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors">
                                        <td className="px-6 py-4 whitespace-nowrap">
                                            <div className="text-sm font-medium text-gray-900 dark:text-white">
                                                <Link to="/problem-environments/$envName" params={{ envName: env.metadata.name }} className="hover:underline text-indigo-600 dark:text-indigo-400">
                                                    {env.metadata.name}
                                                </Link>
                                            </div>
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                                            {env.spec.workerName || <span className="text-gray-400 italic">Unscheduled</span>}
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap">
                                            <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${getStatusColor(env.status)}`}>
                                                {getStatusText(env.status)}
                                            </span>
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                                            <Link to="/problem-environments/$envName" params={{ envName: env.metadata.name }} className="text-indigo-600 hover:text-indigo-900 dark:text-indigo-400 dark:hover:text-indigo-300">
                                                Details
                                            </Link>
                                        </td>
                                    </tr>
                                ))}
                                {paginatedEnvs.length === 0 && (
                                    <tr>
                                        <td colSpan={5} className="px-6 py-10 text-center text-sm text-gray-500 dark:text-gray-400">
                                            No environments found.
                                        </td>
                                    </tr>
                                )}
                            </tbody>
                        </table>
                    </div>
                    <Pagination currentPage={currentPage} totalPages={totalPages} onPageChange={handlePageChange} />
                </div>
            </div>
        </div>
    );
}
