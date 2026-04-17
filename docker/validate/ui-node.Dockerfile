FROM node:20-alpine AS deps
WORKDIR /workspace/ui-node

COPY ui-node/package.json ui-node/package-lock.json ./
RUN npm ci

COPY api /workspace/api
COPY ui-node /workspace/ui-node

FROM deps AS generated-types
RUN npm run gen:openapi

FROM deps AS test
RUN npm run gen:openapi && npm test

FROM deps AS build
RUN npm run gen:openapi && npm run build
