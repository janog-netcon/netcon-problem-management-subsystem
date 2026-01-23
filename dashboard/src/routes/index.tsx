import { createFileRoute, Link } from '@tanstack/react-router';
import { getProblems, getProblemEnvironments, getWorkers } from '../data/k8s';
import { Box, Server, Layers, ArrowRight } from 'lucide-react';

export const Route = createFileRoute('/')({
  component: DashboardHome,
  loader: async () => {
    const [problems, envs, workers] = await Promise.all([
      getProblems(),
      getProblemEnvironments(),
      getWorkers(),
    ]);
    return { problems, envs, workers };
  },
});

function DashboardHome() {
  const { problems, envs, workers } = Route.useLoaderData();

  const problemCount = problems.items.length;
  const envCount = envs.items.length;
  const workerCount = workers.items.length;

  const deployedEnvs = envs.items.filter(e =>
    e.status?.conditions?.some(c => c.type === 'Deployed' && c.status === 'True')
  ).length;

  const readyEnvs = envs.items.filter(e =>
    e.status?.conditions?.some(c => c.type === 'Ready' && c.status === 'True')
  ).length;

  return (
    <div className="p-6 bg-gray-50 dark:bg-gray-900 min-h-screen">
      <div className="max-w-7xl mx-auto space-y-8">
        <header>
          <h1 className="text-3xl font-bold text-gray-900 dark:text-white">NETCON Telescope</h1>
          <p className="mt-2 text-gray-600 dark:text-gray-400">Welcome to the Problem Management Dashboard.</p>
        </header>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          <SummaryCard
            title="Problems"
            count={problemCount}
            icon={<Box className="w-8 h-8 text-white" />}
            color="bg-indigo-500"
            link="/problems"
          />
          <SummaryCard
            title="Environments"
            count={envCount}
            subtext={`${readyEnvs} Ready / ${deployedEnvs} Deployed`}
            icon={<Layers className="w-8 h-8 text-white" />}
            color="bg-teal-500"
            link="/problem-environments"
          />
          <SummaryCard
            title="Workers"
            count={workerCount}
            icon={<Server className="w-8 h-8 text-white" />}
            color="bg-purple-500"
            link="/workers"
          />
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
          <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-lg font-semibold text-gray-900 dark:text-white">Recent Problems</h2>
              <Link to="/problems" search={{ p: 1, q: '' }} className="text-sm text-indigo-600 hover:text-indigo-800 dark:text-indigo-400 flex items-center">
                View all <ArrowRight className="w-4 h-4 ml-1" />
              </Link>
            </div>
            <ul className="divide-y divide-gray-200 dark:divide-gray-700">
              {problems.items.slice(0, 5).map(p => (
                <li key={p.metadata.name} className="py-3">
                  <Link to="/problems/$problemName" params={{ problemName: p.metadata.name }} className="flex justify-between hover:bg-gray-50 dark:hover:bg-gray-700/50 -mx-2 px-2 rounded transition-colors">
                    <span className="text-gray-900 dark:text-white font-medium">{p.metadata.name}</span>
                    <span className="text-sm text-gray-500 dark:text-gray-400">{new Date(p.metadata.creationTimestamp).toLocaleDateString()}</span>
                  </Link>
                </li>
              ))}
              {problems.items.length === 0 && <li className="py-3 text-gray-500 text-sm">No problems found.</li>}
            </ul>
          </div>

          <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-lg font-semibold text-gray-900 dark:text-white">Recent Environments</h2>
              <Link to="/problem-environments" search={{ p: 1, q: '' }} className="text-sm text-indigo-600 hover:text-indigo-800 dark:text-indigo-400 flex items-center">
                View all <ArrowRight className="w-4 h-4 ml-1" />
              </Link>
            </div>
            <ul className="divide-y divide-gray-200 dark:divide-gray-700">
              {envs.items.slice(0, 5).map(e => (
                <li key={e.metadata.name} className="py-3">
                  <Link to="/problem-environments/$envName" params={{ envName: e.metadata.name }} className="flex justify-between hover:bg-gray-50 dark:hover:bg-gray-700/50 -mx-2 px-2 rounded transition-colors">
                    <span className="text-gray-900 dark:text-white font-medium">{e.metadata.name}</span>
                    <span className="text-sm text-xs px-2 py-0.5 rounded bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-300">
                      {e.spec.workerName || 'Pending'}
                    </span>
                  </Link>
                </li>
              ))}
              {envs.items.length === 0 && <li className="py-3 text-gray-500 text-sm">No environments found.</li>}
            </ul>
          </div>
        </div>
      </div>
    </div>
  );
}

function SummaryCard({ title, count, icon, color, subtext, link }: any) {
  const Content = (
    <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6 border-l-4 border-transparent hover:border-indigo-500 transition-all cursor-pointer h-full">
      <div className="flex items-center justify-between">
        <div>
          <p className="text-sm font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">{title}</p>
          <p className="mt-1 text-3xl font-semibold text-gray-900 dark:text-white">{count}</p>
          {subtext && <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{subtext}</p>}
        </div>
        <div className={`p-3 rounded-full ${color} shadow-lg shadow-indigo-500/20`}>
          {icon}
        </div>
      </div>
    </div>
  );

  if (link) {
    // @ts-expect-error: Link path is dynamic, making strict typing difficult for search params
    return <Link to={link} search={{ p: 1, q: '' }} className="block h-full">{Content}</Link>;
  }
  return <div className="h-full">{Content}</div>;
}
