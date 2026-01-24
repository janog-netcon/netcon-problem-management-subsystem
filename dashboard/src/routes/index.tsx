import { createFileRoute, Link } from '@tanstack/react-router';
import { getProblems, getProblemEnvironments, getWorkers } from '../data/k8s';
import { Box, Server, Layers } from 'lucide-react';

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

  const assignedEnvs = envs.items.filter(e =>
    e.status?.conditions?.some(c => c.type === 'Assigned' && c.status === 'True')
  ).length;

  const readyEnvs = envs.items.filter(e => {
    const isReady = e.status?.conditions?.some(c => c.type === 'Ready' && c.status === 'True');
    const isAssigned = e.status?.conditions?.some(c => c.type === 'Assigned' && c.status === 'True');
    return isReady && !isAssigned;
  }).length;

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
            subtext={`${assignedEnvs} Assigned / ${readyEnvs} Ready`}
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
    return <Link to={link} className="block h-full">{Content}</Link>;
  }
  return <div className="h-full">{Content}</div>;
}
