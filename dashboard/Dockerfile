FROM node:25-bookworm-slim AS build

WORKDIR /app

COPY dashboard/package*.json ./
RUN npm install

COPY dashboard/ ./
RUN npm run build

FROM node:25-bookworm-slim

WORKDIR /app

COPY --from=build /app/.output ./.output

EXPOSE 3000

ENV HOST=0.0.0.0
ENV PORT=3000

CMD ["node", ".output/server/index.mjs"]
