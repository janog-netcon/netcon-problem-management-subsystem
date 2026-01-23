import { createServerFn } from '@tanstack/react-start';

// Types are defined locally below to avoid circular dependency or import issues during client bundle generation.

// Re-defining types here to avoid import issues or need for separate file for now.
export interface Problem {
    apiVersion: string;
    kind: string;
    metadata: {
        name: string;
        creationTimestamp: string;
        [key: string]: any;
    };
    spec: {
        template?: any;
        assignableReplicas: number;
        [key: string]: any;
    };
    status?: {
        replicas: {
            total: number;
            scheduled: number;
            assignable: number;
            assigned: number;
        };
    };
}

export interface ProblemList {
    apiVersion: string;
    kind: string;
    metadata: {
        continue?: string;
        resourceVersion?: string;
    };
    items: Problem[];
}

export interface ProblemEnvironment {
    apiVersion: string;
    kind: string;
    metadata: {
        name: string;
        creationTimestamp: string;
        ownerReferences?: any[];
        [key: string]: any;
    };
    spec: {
        topologyFile: any;
        configFiles?: any[];
        workerName?: string;
        workerSelectors?: any[];
    };
    status?: {
        containers?: {
            name: string;
            image: string;
            containerID: string;
            containerName: string;
            ready: boolean;
            managementIPAddress: string;
        }[];
        password?: string;
        conditions?: {
            type: string;
            status: string;
            lastTransitionTime: string;
            reason?: string;
            message?: string;
        }[];
    };
}

export interface ProblemEnvironmentList {
    apiVersion: string;
    kind: string;
    metadata: {
        continue?: string;
        resourceVersion?: string;
    };
    items: ProblemEnvironment[];
}

export interface Worker {
    apiVersion: string;
    kind: string;
    metadata: {
        name: string;
        creationTimestamp: string;
        [key: string]: any;
    };
    spec: {
        disableSchedule: boolean;
    };
    status?: {
        workerInfo: {
            externalIPAddress: string;
            externalPort: number;
            hostname: string;
            memoryUsedPercent: string;
            cpuUsedPercent: string;
        };
        conditions?: {
            type: string;
            status: string;
            lastTransitionTime: string;
            reason?: string;
            message?: string;
        }[];
    };
}

export interface WorkerList {
    apiVersion: string;
    kind: string;
    metadata: {
        continue?: string;
        resourceVersion?: string;
    };
    items: Worker[];
}

const GROUP = 'netcon.janog.gr.jp';
const VERSION = 'v1alpha1';
const NAMESPACE = 'netcon';

// Helper to lazily load the K8s client (Server Side Only)
// This ensures that @kubernetes/client-node is not included in the client bundle.
async function getApiClient() {
    const { KubeConfig, CustomObjectsApi } = await import('@kubernetes/client-node');
    const kc = new KubeConfig();
    kc.loadFromDefault();
    return kc.makeApiClient(CustomObjectsApi);
}

// --- Server Functions ---

export const getProblems = createServerFn({ method: "GET" })
    .handler(async () => {
        try {
            const customApi = await getApiClient();
            const res = await customApi.listNamespacedCustomObject({
                group: GROUP,
                version: VERSION,
                plural: 'problems',
                namespace: NAMESPACE,
            });
            return res as ProblemList;
        } catch (err) {
            console.error('Failed to fetch problems:', err);
            throw err;
        }
    });

export const getProblem = createServerFn({ method: "GET" })
    .inputValidator((name: string) => name)
    .handler(async ({ data: name }) => {
        try {
            const customApi = await getApiClient();
            const res = await customApi.getNamespacedCustomObject({
                group: GROUP,
                version: VERSION,
                plural: 'problems',
                namespace: NAMESPACE,
                name: name,
            });
            return res as Problem;
        } catch (err) {
            console.error(`Failed to fetch problem ${name}:`, err);
            throw err;
        }
    });

export const getProblemEnvironments = createServerFn({ method: "GET" })
    .handler(async () => {
        try {
            const customApi = await getApiClient();
            const res = await customApi.listNamespacedCustomObject({
                group: GROUP,
                version: VERSION,
                plural: 'problemenvironments',
                namespace: NAMESPACE,
            });
            return res as ProblemEnvironmentList;
        } catch (err) {
            console.error('Failed to fetch problem environments:', err);
            throw err;
        }
    });

export const getProblemEnvironment = createServerFn({ method: "GET" })
    .inputValidator((name: string) => name)
    .handler(async ({ data: name }) => {
        try {
            const customApi = await getApiClient();
            const res = await customApi.getNamespacedCustomObject({
                group: GROUP,
                version: VERSION,
                plural: 'problemenvironments',
                namespace: NAMESPACE,
                name: name,
            });
            return res as ProblemEnvironment;
        } catch (err) {
            console.error(`Failed to fetch problem environment ${name}:`, err);
            throw err;
        }
    });

export const getWorkers = createServerFn({ method: "GET" })
    .handler(async () => {
        try {
            const customApi = await getApiClient();
            const res = await customApi.listClusterCustomObject({
                group: GROUP,
                version: VERSION,
                plural: 'workers',
            });
            return res as WorkerList;
        } catch (err) {
            console.error('Failed to fetch workers:', err);
            throw err;
        }
    });

export const getWorker = createServerFn({ method: "GET" })
    .inputValidator((name: string) => name)
    .handler(async ({ data: name }) => {
        try {
            const customApi = await getApiClient();
            const res = await customApi.getClusterCustomObject({
                group: GROUP,
                version: VERSION,
                plural: 'workers',
                name: name,
            });
            return res as Worker;
        } catch (err) {
            console.error(`Failed to fetch worker ${name}:`, err);
            throw err;
        }
    });
