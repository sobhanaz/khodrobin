FROM node:22-alpine AS build
WORKDIR /app
# pnpm, not npm: npm's tree resolver crashes on the current Nuxt dependency
# graph with "Cannot read properties of null (reading 'edgesOut')" — reproduced
# with npm 10 and 11, and with nuxt+vue alone. pnpm is also what Nuxt recommends.
RUN npm i -g pnpm@9
COPY package.json pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY . .
RUN pnpm run build

FROM node:22-alpine
WORKDIR /app
ENV NODE_ENV=production
ENV NITRO_PORT=3000
ENV NITRO_HOST=0.0.0.0
# Only the Nitro output is needed at runtime; dependencies are already bundled.
COPY --from=build /app/.output ./.output
EXPOSE 3000
USER node
CMD ["node", ".output/server/index.mjs"]
