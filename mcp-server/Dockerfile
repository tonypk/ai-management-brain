# Stage 1: Build
FROM node:20-alpine AS build
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY tsconfig.json ./
COPY src ./src
RUN npx tsc

# Stage 2: Production
FROM node:20-alpine
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci --omit=dev && npm cache clean --force
COPY --from=build /app/dist ./dist

ENV NODE_ENV=production
ENV TRANSPORT=http
ENV MCP_PORT=3100
EXPOSE 3100

USER node
CMD ["node", "dist/index.js", "--http"]
