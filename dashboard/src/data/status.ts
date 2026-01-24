import { ProblemEnvironment } from './k8s';

export const getStatusColor = (status: ProblemEnvironment['status']) => {
    const isReady = status?.conditions?.find(c => c.type === 'Ready' && c.status === 'True');
    const isAssigned = status?.conditions?.find(c => c.type === 'Assigned' && c.status === 'True');

    if (isAssigned) return 'bg-indigo-100 text-indigo-800 dark:bg-indigo-900/50 dark:text-indigo-300';
    if (isReady) return 'bg-green-100 text-green-800 dark:bg-green-900/50 dark:text-green-300';

    return 'bg-blue-100 text-blue-800 dark:bg-blue-900/50 dark:text-blue-300';
};

export const getStatusText = (status: ProblemEnvironment['status']) => {
    const isReady = status?.conditions?.find(c => c.type === 'Ready' && c.status === 'True');
    const isAssigned = status?.conditions?.find(c => c.type === 'Assigned' && c.status === 'True');

    if (isAssigned) return 'Assigned';
    if (isReady) return 'Ready';
    return 'Deploying';
};
