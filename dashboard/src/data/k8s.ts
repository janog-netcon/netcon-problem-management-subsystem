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

export const getDeploymentLog = createServerFn({ method: "GET" })
    .inputValidator((envName: string) => envName)
    .handler(async ({ data: envName }) => {
        try {
            const { KubeConfig, CoreV1Api } = await import('@kubernetes/client-node');
            const kc = new KubeConfig();
            kc.loadFromDefault();
            const k8sApi = kc.makeApiClient(CoreV1Api);

            // List all ConfigMaps in the namespace
            // NOTE: fieldSelector for name is not supported for partial matches, so we list and filter.
            const res = await k8sApi.listNamespacedConfigMap({
                namespace: NAMESPACE,
            });

            const prefix = `deploy-${envName}`;
            // Find the ConfigMap that starts with the prefix and has stderr data
            // We search for any matching ConfigMap, even if it doesn't have data, just to be safe,
            // but the original logic filtered for stderr. Let's filter for either stdout or stderr.
            const cm = res.items.find((item: any) =>
                item.metadata?.name?.startsWith(prefix)
            );

            if (cm && cm.data) {
                return {
                    stdout: cm.data['stdout'] || null,
                    stderr: cm.data['stderr'] || null
                };
            }
            return { stdout: null, stderr: null };
            // Return null instead of throwing so the page can still load
            return null;
        } catch (err) {
            console.error(`Failed to fetch deployment log for ${envName}:`, err);
            // Return null instead of throwing so the page can still load
            return null;
        }
    });
export const deleteProblemEnvironment = createServerFn({ method: "POST" })
    .inputValidator((name: string) => name)
    .handler(async ({ data: name }) => {
        try {
            const customApi = await getApiClient();
            await customApi.deleteNamespacedCustomObject({
                group: GROUP,
                version: VERSION,
                namespace: NAMESPACE,
                plural: 'problemenvironments',
                name: name
            });
            return { success: true };
        } catch (err: any) {
            console.error(`Failed to delete problem environment ${name}:`, err);
            throw err;
        }
    });

export const assignProblemEnvironment = createServerFn({ method: "POST" })
    .inputValidator((name: string) => name)
    .handler(async ({ data: name }) => {
        try {
            const customApi = await getApiClient();
            const env = await customApi.getNamespacedCustomObject({
                group: GROUP,
                version: VERSION,
                namespace: NAMESPACE,
                plural: 'problemenvironments',
                name: name,
            }) as ProblemEnvironment;

            const conditions = env.status?.conditions || [];
            const now = new Date().toISOString();

            // Find or create Assigned condition
            const assignedIdx = conditions.findIndex((c: any) => c.type === 'Assigned');
            const newCondition = {
                type: 'Assigned',
                status: 'True',
                lastTransitionTime: now,
                reason: 'AdminUpdated',
                message: 'assigned by admin forcibly'
            };

            if (assignedIdx >= 0) {
                conditions[assignedIdx] = newCondition;
            } else {
                conditions.push(newCondition);
            }

            // Using JSON Patch (array of ops) instead of Merge Patch because the
            // library defaults to 'application/json-patch+json' and the server
            // expects an array for that content type.
            await customApi.patchNamespacedCustomObjectStatus({
                group: GROUP,
                version: VERSION,
                namespace: NAMESPACE,
                plural: 'problemenvironments',
                name: name,
                body: [
                    {
                        op: 'replace',
                        path: '/status',
                        value: {
                            ...(env.status || {}),
                            conditions: conditions
                        }
                    }
                ]
            });

            return { success: true };
        } catch (err: any) {
            console.error(`Failed to assign problem environment ${name}:`, err);
            throw err;
        }
    });

export const unassignProblemEnvironment = createServerFn({ method: "POST" })
    .inputValidator((name: string) => name)
    .handler(async ({ data: name }) => {
        try {
            const customApi = await getApiClient();
            const env = await customApi.getNamespacedCustomObject({
                group: GROUP,
                version: VERSION,
                namespace: NAMESPACE,
                plural: 'problemenvironments',
                name: name,
            }) as ProblemEnvironment;

            const conditions = env.status?.conditions || [];
            const now = new Date().toISOString();

            // Find or create Assigned condition
            const assignedIdx = conditions.findIndex((c: any) => c.type === 'Assigned');
            const newCondition = {
                type: 'Assigned',
                status: 'False',
                lastTransitionTime: now,
                reason: 'AdminUpdated',
                message: 'unassigned by admin forcibly'
            };

            if (assignedIdx >= 0) {
                conditions[assignedIdx] = newCondition;
            } else {
                conditions.push(newCondition);
            }

            // Using JSON Patch (array of ops) instead of Merge Patch because the
            // library defaults to 'application/json-patch+json' and the server
            // expects an array for that content type.
            await customApi.patchNamespacedCustomObjectStatus({
                group: GROUP,
                version: VERSION,
                namespace: NAMESPACE,
                plural: 'problemenvironments',
                name: name,
                body: [
                    {
                        op: 'replace',
                        path: '/status',
                        value: {
                            ...(env.status || {}),
                            conditions: conditions
                        }
                    }
                ]
            });

            return { success: true };
        } catch (err: any) {
            console.error(`Failed to unassign problem environment ${name}:`, err);
            throw err;
        }
    });

export const updateWorkerSchedule = createServerFn({ method: "POST" })
    .inputValidator((data: { name: string; disabled: boolean }) => data)
    .handler(async ({ data: { name, disabled } }) => {
        try {
            const customApi = await getApiClient();
            // Using JSON Patch for update
            await customApi.patchClusterCustomObject({
                group: GROUP,
                version: VERSION,
                plural: 'workers',
                name: name,
                body: [
                    {
                        op: 'replace',
                        path: '/spec/disableSchedule',
                        value: disabled
                    }
                ]
            });
            return { success: true };
        } catch (err: any) {
            throw err;
        }
    });

export const getConfigMap = createServerFn({ method: "GET" })
    .inputValidator((name: string) => name)
    .handler(async ({ data: name }) => {
        try {
            const { KubeConfig, CoreV1Api } = await import('@kubernetes/client-node');
            const kc = new KubeConfig();
            kc.loadFromDefault();
            const k8sApi = kc.makeApiClient(CoreV1Api);

            const res = await k8sApi.readNamespacedConfigMap({
                name: name,
                namespace: NAMESPACE,
            });
            // Convert to plain object to handle serialization
            return JSON.parse(JSON.stringify(res));
        } catch (err) {
            console.error(`Failed to fetch configmap ${name}:`, err);
            // Return null if not found or error, so UI can handle it gracefully (e.g. skip tab)
            return null;
        }
    });
