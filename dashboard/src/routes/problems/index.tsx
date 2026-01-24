import { createFileRoute, Link, useNavigate, useRouter } from '@tanstack/react-router';
import { getProblems } from '../../data/k8s';
import { SearchBar } from '../../components/SearchBar';
import { z } from 'zod';
import { ArrowUp, ArrowDown, ArrowUpDown, RefreshCw } from 'lucide-react';

const problemSearchSchema = z.object({
    q: z.string().optional(),
    sort: z.enum(['desired', 'ready', 'assigned', 'deploying', 'total']).optional(),
    dir: z.enum(['asc', 'desc']).optional(),
});

export const Route = createFileRoute('/problems/')({
    component: ProblemsPage,
    loader: async () => {
        const problems = await getProblems();
        return { problems, updatedAt: new Date() };
    },
    validateSearch: (search) => problemSearchSchema.parse(search),
    staleTime: 60000, // 1 minute
    gcTime: 300000,   // 5 minutes
    shouldReload: false,
});

function ProblemsPage() {
    const { problems, updatedAt } = Route.useLoaderData();
    const search = Route.useSearch();
    const navigate = useNavigate({ from: Route.fullPath });
    const router = useRouter();


    // Client-side filtering
    const filteredProblems = problems.items.filter((problem) => {
        if (!search.q) return true;
        return problem.metadata.name.toLowerCase().includes(search.q.toLowerCase());
    });

    // Client-side sorting
    const sortedProblems = [...filteredProblems].sort((a, b) => {
        if (!search.sort) return 0;

        const dir = search.dir === 'desc' ? -1 : 1;

        const getVal = (p: typeof a) => {
            switch (search.sort) {
                case 'desired': return p.spec.assignableReplicas;
                case 'ready': return p.status?.replicas?.assignable ?? 0;
                case 'assigned': return p.status?.replicas?.assigned ?? 0;
                case 'deploying': return (p.status?.replicas?.total ?? 0) - ((p.status?.replicas?.assignable ?? 0) + (p.status?.replicas?.assigned ?? 0));
                case 'total': return p.status?.replicas?.total ?? 0;
                default: return 0;
            }
        };

        const valA = getVal(a);
        const valB = getVal(b);

        if (valA < valB) return -1 * dir;
        if (valA > valB) return 1 * dir;
        return 0;
    });



    const handleSearch = (newQuery: string) => {
        navigate({
            search: (prev) => ({ ...prev, q: newQuery || undefined }),
            replace: true,
        });
    };

    const handleSort = (column: 'desired' | 'ready' | 'assigned' | 'deploying' | 'total') => {
        navigate({
            search: (prev) => {
                if (prev.sort === column) {
                    if (prev.dir === 'asc') return { ...prev, dir: 'desc' };
                    return { ...prev, sort: undefined, dir: undefined };
                }
                return { ...prev, sort: column, dir: 'asc' };
            }
        });
    };



    return (
        <div className="p-6 bg-gray-50 dark:bg-gray-900 min-h-screen">
            <div className="max-w-7xl mx-auto">
                <div className="flex flex-col md:flex-row md:items-center md:justify-between mb-8">
                    <div>
                        <h1 className="text-3xl font-bold text-gray-900 dark:text-white">Problems</h1>
                        <p className="mt-2 text-sm text-gray-600 dark:text-gray-400">
                            Manage and view the status of deployed problems.
                        </p>
                    </div>
                    <div className="mt-4 md:mt-0 flex flex-col items-end space-y-2">
                        <div className="flex items-center space-x-3 w-full md:w-auto">
                            <div className="w-full md:w-64">
                                <SearchBar value={search.q || ''} onChange={handleSearch} placeholder="Search problems..." />
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
                                    <th
                                        scope="col"
                                        className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider"
                                    >
                                        Name
                                    </th>
                                    <th
                                        scope="col"
                                        className="px-4 py-3 text-center text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider w-24 cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-600 transition-colors"
                                        onClick={() => handleSort('desired')}
                                    >
                                        <div className="flex items-center justify-center space-x-1">
                                            <span>Desired</span>
                                            <SortIcon column="desired" search={search} />
                                        </div>
                                    </th>
                                    <th
                                        scope="col"
                                        className="px-4 py-3 text-center text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider w-24 cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-600 transition-colors"
                                        onClick={() => handleSort('ready')}
                                    >
                                        <div className="flex items-center justify-center space-x-1">
                                            <span>Ready</span>
                                            <SortIcon column="ready" search={search} />
                                        </div>
                                    </th>
                                    <th
                                        scope="col"
                                        className="px-4 py-3 text-center text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider w-24 cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-600 transition-colors"
                                        onClick={() => handleSort('assigned')}
                                    >
                                        <div className="flex items-center justify-center space-x-1">
                                            <span>Assigned</span>
                                            <SortIcon column="assigned" search={search} />
                                        </div>
                                    </th>
                                    <th
                                        scope="col"
                                        className="px-4 py-3 text-center text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider w-24 cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-600 transition-colors"
                                        onClick={() => handleSort('deploying')}
                                    >
                                        <div className="flex items-center justify-center space-x-1">
                                            <span>Deploying</span>
                                            <SortIcon column="deploying" search={search} />
                                        </div>
                                    </th>
                                    <th
                                        scope="col"
                                        className="px-4 py-3 text-center text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider w-24 cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-600 transition-colors"
                                        onClick={() => handleSort('total')}
                                    >
                                        <div className="flex items-center justify-center space-x-1">
                                            <span>Total</span>
                                            <SortIcon column="total" search={search} />
                                        </div>
                                    </th>
                                    <th scope="col" className="px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider w-24">
                                        Action
                                    </th>
                                </tr>
                            </thead>
                            <tbody className="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
                                {sortedProblems.map((problem) => (
                                    <tr key={problem.metadata.name} className="hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors">
                                        <td className="px-6 py-4 whitespace-nowrap">
                                            <div className="text-sm font-medium text-gray-900 dark:text-white">
                                                <Link to="/problems/$problemName" params={{ problemName: problem.metadata.name }} className="hover:underline text-indigo-600 dark:text-indigo-400">
                                                    {problem.metadata.name}
                                                </Link>
                                            </div>
                                        </td>
                                        <td className="px-4 py-4 whitespace-nowrap text-center">
                                            <span className="text-sm font-medium text-gray-900 dark:text-white bg-gray-100 dark:bg-gray-700 px-2.5 py-1 rounded">
                                                {problem.spec.assignableReplicas}
                                            </span>
                                        </td>
                                        <td className="px-4 py-4 whitespace-nowrap text-center">
                                            <span className="text-sm font-medium text-green-700 dark:text-green-400 bg-green-100 dark:bg-green-900/30 px-2.5 py-1 rounded">
                                                {problem.status?.replicas?.assignable ?? 0}
                                            </span>
                                        </td>
                                        <td className="px-4 py-4 whitespace-nowrap text-center">
                                            <span className="text-sm font-medium text-indigo-700 dark:text-indigo-400 bg-indigo-100 dark:bg-indigo-900/30 px-2.5 py-1 rounded">
                                                {problem.status?.replicas?.assigned ?? 0}
                                            </span>
                                        </td>
                                        <td className="px-4 py-4 whitespace-nowrap text-center">
                                            <span className="text-sm font-medium text-blue-700 dark:text-blue-400 bg-blue-100 dark:bg-blue-900/30 px-2.5 py-1 rounded">
                                                {(problem.status?.replicas?.total ?? 0) - ((problem.status?.replicas?.assignable ?? 0) + (problem.status?.replicas?.assigned ?? 0))}
                                            </span>
                                        </td>
                                        <td className="px-4 py-4 whitespace-nowrap text-center">
                                            <span className="text-sm font-medium text-purple-700 dark:text-purple-400 bg-purple-100 dark:bg-purple-900/30 px-2.5 py-1 rounded">
                                                {problem.status?.replicas?.total ?? 0}
                                            </span>
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                                            <Link to="/problems/$problemName" params={{ problemName: problem.metadata.name }} className="text-indigo-600 hover:text-indigo-900 dark:text-indigo-400 dark:hover:text-indigo-300">
                                                Details
                                            </Link>
                                        </td>
                                    </tr>
                                ))}
                                {sortedProblems.length === 0 && (
                                    <tr>
                                        <td colSpan={6} className="px-6 py-10 text-center text-sm text-gray-500 dark:text-gray-400">
                                            No problems found.
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

function SortIcon({ column, search }: { column: string, search: any }) {
    if (search.sort !== column) return <ArrowUpDown className="w-3 h-3 opacity-30" />;
    return search.dir === 'asc'
        ? <ArrowUp className="w-3 h-3 text-indigo-500" />
        : <ArrowDown className="w-3 h-3 text-indigo-500" />;
}
