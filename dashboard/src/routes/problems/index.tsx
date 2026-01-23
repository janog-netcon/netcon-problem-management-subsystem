import { createFileRoute, Link, useNavigate } from '@tanstack/react-router';
import { getProblems, Problem } from '../../data/k8s';
import { SearchBar } from '../../components/SearchBar';
import { Pagination } from '../../components/Pagination';
import { z } from 'zod';

const problemSearchSchema = z.object({
    p: z.number().default(1),
    q: z.string().default(''),
});

export const Route = createFileRoute('/problems/')({
    component: ProblemsPage,
    loader: async () => await getProblems(),
    validateSearch: (search) => problemSearchSchema.parse(search),
});

function ProblemsPage() {
    const problemsList = Route.useLoaderData();
    const search = Route.useSearch();
    const navigate = useNavigate({ from: Route.fullPath });

    const itemsPerPage = 20;

    // Client-side filtering
    const filteredProblems = problemsList.items.filter((problem) => {
        if (!search.q) return true;
        return problem.metadata.name.toLowerCase().includes(search.q.toLowerCase());
    });

    const totalItems = filteredProblems.length;
    const totalPages = Math.ceil(totalItems / itemsPerPage);

    // Ensure current page is valid
    const currentPage = Math.min(Math.max(1, search.p), Math.max(1, totalPages));

    const paginatedProblems = filteredProblems.slice(
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
                    <div className="mt-4 md:mt-0 w-full md:w-64">
                        <SearchBar value={search.q} onChange={handleSearch} placeholder="Search problems..." />
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
                                        Replicas (Total/Sched/Rdy/Asgn)
                                    </th>
                                    <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                                        Age
                                    </th>
                                    <th scope="col" className="relative px-6 py-3">
                                        <span className="sr-only">View</span>
                                    </th>
                                </tr>
                            </thead>
                            <tbody className="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
                                {paginatedProblems.map((problem) => (
                                    <tr key={problem.metadata.name} className="hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors">
                                        <td className="px-6 py-4 whitespace-nowrap">
                                            <div className="text-sm font-medium text-gray-900 dark:text-white">
                                                <Link to="/problems/$problemName" params={{ problemName: problem.metadata.name }} className="hover:underline text-indigo-600 dark:text-indigo-400">
                                                    {problem.metadata.name}
                                                </Link>
                                            </div>
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap">
                                            <div className="text-sm text-gray-500 dark:text-gray-400 flex space-x-2">
                                                <span title="Total" className="px-2 py-1 rounded bg-gray-100 dark:bg-gray-700 text-gray-800 dark:text-gray-200">{problem.status?.replicas?.total ?? 0}</span>
                                                <span>/</span>
                                                <span title="Scheduled" className="px-2 py-1 rounded bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-200">{problem.status?.replicas?.scheduled ?? 0}</span>
                                                <span>/</span>
                                                <span title="Assignable (Ready)" className="px-2 py-1 rounded bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-200">{problem.status?.replicas?.assignable ?? 0}</span>
                                                <span>/</span>
                                                <span title="Assigned" className="px-2 py-1 rounded bg-purple-100 dark:bg-purple-900/30 text-purple-800 dark:text-purple-200">{problem.status?.replicas?.assigned ?? 0}</span>
                                            </div>
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                                            {new Date(problem.metadata.creationTimestamp).toLocaleDateString()}
                                        </td>
                                        <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                                            <Link to="/problems/$problemName" params={{ problemName: problem.metadata.name }} className="text-indigo-600 hover:text-indigo-900 dark:text-indigo-400 dark:hover:text-indigo-300">
                                                Details
                                            </Link>
                                        </td>
                                    </tr>
                                ))}
                                {paginatedProblems.length === 0 && (
                                    <tr>
                                        <td colSpan={5} className="px-6 py-10 text-center text-sm text-gray-500 dark:text-gray-400">
                                            No problems found.
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
