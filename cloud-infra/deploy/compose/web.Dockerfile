FROM node:22-alpine AS build
WORKDIR /src
COPY web/package*.json ./
RUN npm ci
COPY web/index.html ./index.html
COPY web/tsconfig.json ./tsconfig.json
COPY web/src ./src
COPY web/public ./public
RUN npm run build

FROM nginx:1.29-alpine
COPY deploy/compose/web.nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /src/dist /usr/share/nginx/html
