import { useEffect, useRef, useState } from 'react';

import { graphviz } from 'd3-graphviz';
import { Card } from './Card';
import { Activity, AlertTriangle } from 'lucide-react';
import yaml from 'js-yaml';

interface TopologyViewerProps {
    manifestContent: string;
}

interface ManifestTopology {
    nodes?: Record<string, { kind?: string; image?: string;[key: string]: any }>;
    links?: { endpoints: string[] }[];
}

export function TopologyViewer({ manifestContent }: TopologyViewerProps) {
    const graphRef = useRef<HTMLDivElement>(null);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        const renderTopology = async () => {
            setError(null);
            try {
                const parsed = yaml.load(manifestContent) as any;
                const topology = parsed?.topology as ManifestTopology;

                if (!topology || !topology.nodes) {
                    setError('No topology found in manifest.');
                    return;
                }

                let dotLines = [];
                dotLines.push('bgcolor="transparent";');
                dotLines.push('rankdir=LR;');
                dotLines.push('nodesep=2.0;');
                dotLines.push('ranksep=1.5;');
                // Dark mode styles: white text, light blue/indigo nodes, light gray edges
                dotLines.push('node [shape=rect, style="filled,rounded", fillcolor="#1e1e2e", color="#89b4fa", penwidth=2, fontname="sans-serif", fontcolor="#cdd6f4", margin=0.2];');
                dotLines.push('edge [fontname="sans-serif", fontsize=10, fontcolor="#a6adc8", color="#585b70"];');

                Object.entries(topology.nodes).forEach(([name, _]) => {
                    if (name !== "bridges") {
                        dotLines.push(`"${name}" [label="${name}"];`);
                    }
                });

                if (topology.links) {
                    topology.links.forEach((link) => {
                        if (link.endpoints && link.endpoints.length >= 2) {
                            const [srcStr, tgtStr] = link.endpoints;

                            const [srcNode, srcInt] = srcStr.split(':');
                            const [tgtNode, tgtInt] = tgtStr.split(':');

                            if (srcNode && tgtNode) {
                                const tailLabel = srcInt || '';
                                const headLabel = tgtInt || '';
                                dotLines.push(`"${srcNode}" -- "${tgtNode}" [taillabel="${tailLabel}", headlabel="${headLabel}", labeldistance=2.0];`);
                            }
                        }
                    });
                }

                const dotString = `strict graph G {\n${dotLines.join('\n')}\n}`;

                if (graphRef.current) {
                    const graph = graphviz(graphRef.current)
                        .width(graphRef.current.clientWidth)
                        .height(graphRef.current.clientHeight)
                        .fit(true)
                        .zoom(true)

                    graph.renderDot(dotString)
                        .on('end', () => {
                            // d3-graphviz acts on the SVG.
                            // Accessing the zoom behavior is tricky without the d3 selection referencing the zoom behavior directly.
                            // But we can just rely on fit(true) to start.
                            // User asked to prevent moving image out of frame.
                            // This usually requires setting translateExtent on the d3 zoom behavior.
                            // For now, let's address the visual/label requests first.
                        });
                }

            } catch (e: any) {
                console.error('Graphviz rendering failed:', e);
                setError(`Failed to render topology: ${e.message}`);
            }
        };

        renderTopology();
    }, [manifestContent]);

    if (error) {
        return (
            <Card title={<><Activity className="w-5 h-5 mr-2" /> Network Topology</>}>
                <div className="p-4 bg-red-50 dark:bg-red-900/20 text-red-700 dark:text-red-400 rounded-md flex items-center">
                    <AlertTriangle className="w-5 h-5 mr-2" />
                    {error}
                </div>
            </Card>
        );
    }

    return (
        <Card title={<><Activity className="w-5 h-5 mr-2" /> Network Topology</>}>
            <div className="w-full h-[500px] bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-700 rounded-lg overflow-hidden">
                <div ref={graphRef} className="w-full h-full" style={{ textAlign: 'center' }} />
            </div>
        </Card>
    );
}
