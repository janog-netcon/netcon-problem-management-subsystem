import { createFileRoute, Link, useNavigate, useRouter } from '@tanstack/react-router';
import { getProblemEnvironments, getProblems, getWorkers } from '../../data/k8s';
import { getStatusColor, getStatusText } from '../../data/status';
import { SearchBar } from '../../components/SearchBar';
import { Pagination } from '../../components/Pagination';
import { MultiSelect } from '../../components/MultiSelect';
import { z } from 'zod';
import { RefreshCw } from 'lucide-react';

const envSearchSchema = z.object({
    p: z.number().optional(),
    q: z.string().optional(),
    worker: z.preprocess((v) => (typeof v === 'string' ? v.split(',').filter(Boolean) : v), z.array(z.string())).optional(),
    problem: z.preprocess((v) => (typeof v === 'string' ? v.split(',').filter(Boolean) : v), z.array(z.string())).optional(),
    status: z.preprocess((v) => (typeof v === 'string' ? v.split(',').filter(Boolean) : v), z.array(z.string())).optional(),
});

export const Route = createFileRoute('/problem-environments/')({
    component: ProblemEnvironmentsPage,
    loader: async () => {
        const [envs, problems, workers] = await Promise.all([
            getProblemEnvironments(),
            getProblems(),
            getWorkers(),
        ]);
        return { envs, problems, workers, updatedAt: new Date() };
    },
    validateSearch: (search) => envSearchSchema.parse(search),
    staleTime: 60000,
    gcTime: 300000,
    shouldReload: false,
});


function ProblemEnvironmentsPage() {
    const { envs, problems, workers, updatedAt } = Route.useLoaderData();
    const search = Route.useSearch();
    const navigate = useNavigate({ from: Route.fullPath });
    const router = useRouter();


    const itemsPerPage = 20;

    const filteredEnvs = envs.items.filter((env) => {
        // Name search (q)
        if (search.q && !env.metadata.name.toLowerCase().includes(search.q.toLowerCase())) {
            return false;
        }

        // Worker filter
        if (search.worker && search.worker.length > 0) {
            if (!search.worker.includes(env.spec.workerName || '')) {
                return false;
            }
        }

        // Problem filter
        if (search.problem && search.problem.length > 0) {
            const parentProblem = env.metadata.ownerReferences?.find(ref => ref.kind === 'Problem')?.name;
            if (!search.problem.includes(parentProblem || '')) {
                return false;
            }
        }

        // Status filter
        if (search.status && search.status.length > 0) {
            const statusText = getStatusText(env.status);
            if (!search.status.includes(statusText)) {
                return false;
            }
        }

        return true;
    });

    const totalItems = filteredEnvs.length;
    const totalPages = Math.ceil(totalItems / itemsPerPage);
    const currentPage = Math.min(Math.max(1, search.p ?? 1), Math.max(1, totalPages));

    const paginatedEnvs = filteredEnvs.slice(
        (currentPage - 1) * itemsPerPage,
        currentPage * itemsPerPage
    );

    const handleSearch = (newQuery: string) => {
        navigate({
            search: (prev) => ({ ...prev, q: newQuery || undefined, p: undefined }),
            replace: true,
        });
    };

    const handleFilterChange = (key: 'worker' | 'problem' | 'status', value: string) => {
        navigate({
            search: (prev) => {
                const current = prev[key] || [];
                const updated = current.includes(value)
                    ? current.filter(v => v !== value)
                    : [...current, value];
                return { ...prev, [key]: updated.length > 0 ? updated : undefined, p: undefined };
            },
            replace: true,
        });
    };

    const handlePageChange = (newPage: number) => {
        navigate({
            search: (prev) => ({ ...prev, p: newPage === 1 ? undefined : newPage }),
        });
    };

    return (
        <div className="p-6 bg-gray-50 dark:bg-gray-900 min-h-screen">
            <div className="max-w-7xl mx-auto">
                <div className="flex flex-col md:flex-row md:items-start md:justify-between mb-8 gap-4">
                    <div className="flex-grow">
                        <h1 className="text-3xl font-bold text-gray-900 dark:text-white">Problem Environments</h1>
                        <p className="mt-2 text-sm text-gray-600 dark:text-gray-400">View and manage problem environments running on workers.</p>

                        {/* Filters */}
                        <div className="mt-6 flex flex-wrap gap-4 items-end">
                            <MultiSelect
                                label="Worker"
                                options={[
                                    ...workers.items.map(w => ({ label: w.metadata.name, value: w.metadata.name })),
                                    { label: 'Unscheduled', value: '' }
                                ]}
                                selectedValues={search.worker || []}
                                onChange={(value) => handleFilterChange('worker', value)}
                                placeholder="All Workers"
                            />

                            <MultiSelect
                                label="Problem"
                                options={problems.items.map(p => ({ label: p.metadata.name, value: p.metadata.name }))}
                                selectedValues={search.problem || []}
                                onChange={(value) => handleFilterChange('problem', value)}
                                placeholder="All Problems"
                            />

                            <MultiSelect
                                label="Status"
                                options={['Assigned', 'Ready', 'Deploying'].map(s => ({ label: s, value: s }))}
                                selectedValues={search.status || []}
                                onChange={(value) => handleFilterChange('status', value)}
                                placeholder="All Statuses"
                            />
                        </div>
                    </div>
                    <div className="mt-4 md:mt-0 flex flex-col items-end space-y-2 shrink-0">
                        <div className="flex items-center space-x-3 w-full md:w-auto">
                            <div className="w-full md:w-64">
                                <SearchBar value={search.q || ''} onChange={handleSearch} placeholder="Search environments..." />
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
