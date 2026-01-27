import { createRouter } from '@tanstack/react-router'

// Import the generated route tree
import { routeTree } from './routeTree.gen'

// Create a new router instance
export const getRouter = () => {
  const router = createRouter({
    routeTree,
    context: {},
    parseSearch: (search) => {
      const params = new URLSearchParams(search);
      const res: any = {};
      params.forEach((value, key) => {
        res[key] = value;
      });
      return res;
    },
    stringifySearch: (search) => {
      const params = new URLSearchParams();
      Object.entries(search).forEach(([key, value]) => {
        if (value === undefined || value === null) return;
        if (Array.isArray(value)) {
          if (value.length > 0) {
            params.set(key, value.filter(Boolean).join(','));
          }
        } else {
          params.set(key, String(value));
        }
      });
      const res = params.toString();
      return res ? `?${res}` : '';
    },
    scrollRestoration: true,
    defaultPreloadStaleTime: 0,
  })

  return router
}
