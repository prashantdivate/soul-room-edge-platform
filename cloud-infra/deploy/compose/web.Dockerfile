FROM node:24.21.0-alpine3.24 AS build
WORKDIR /src
COPY web/package*.json ./
RUN npm ci
COPY web/index.html ./index.html
COPY web/tsconfig.json ./tsconfig.json
COPY web/src ./src
COPY web/public ./public
RUN npm run build

FROM nginx:1.31.6-alpine3.24-slim
COPY deploy/compose/web.nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /src/dist /usr/share/nginx/html
RUN chown -R nginx:nginx /usr/share/nginx/html /var/cache/nginx /etc/nginx/conf.d && \
    touch /var/run/nginx.pid && \
    chown nginx:nginx /var/run/nginx.pid
USER nginx
