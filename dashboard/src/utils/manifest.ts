export function cleanManifest(obj: any): any {
    if (!obj || typeof obj !== 'object') return obj;

    const cleaned = JSON.parse(JSON.stringify(obj));

    if (cleaned.metadata) {
        delete cleaned.metadata.uid;
        delete cleaned.metadata.resourceVersion;
        delete cleaned.metadata.generation;
        delete cleaned.metadata.creationTimestamp;
        delete cleaned.metadata.managedFields;
        delete cleaned.metadata.selfLink;

        if (cleaned.metadata.annotations) {
            delete cleaned.metadata.annotations['kubectl.kubernetes.io/last-applied-configuration'];
            delete cleaned.metadata.annotations['argocd.argoproj.io/tracking-id'];
            if (Object.keys(cleaned.metadata.annotations).length === 0) {
                delete cleaned.metadata.annotations;
            }
        }
    }

    // Optionally remove status if we only want "Manifest" as in "Spec",
    // but usually in these dashboards people want to see the status too but in YAML.
    // The user didn't explicitly say to remove status, just "managed fields".

    return cleaned;
}
