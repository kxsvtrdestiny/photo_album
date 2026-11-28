FROM node:22-alpine AS build
WORKDIR /src
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/index.html frontend/vite.config.js ./
COPY frontend/src ./src
RUN npm run build

FROM nginx:1.28-alpine
COPY --from=build /src/dist /usr/share/nginx/html
COPY deploy/nginx.conf /tmp/photo-album.conf
RUN sed -e 's/\r$//' \
        -e 's|__PHOTO_ALBUM_DIST__|/usr/share/nginx/html|' \
        -e 's|listen 127.0.0.1:8088;|listen 8088;|' \
        -e 's|proxy_pass http://127.0.0.1:8080;|proxy_pass http://album_backend;|' \
        -e 's|logs/photo-album-error.log|/dev/stderr|' \
        -e 's|logs/photo-album-access.log|/dev/stdout|' \
        -e 's|logs/photo-album.pid|/var/run/nginx.pid|' \
        -e '/_temp_path /d' \
        -e '/^http {/a\    resolver 127.0.0.11 valid=10s ipv6=off;\n    upstream album_backend {\n        zone album_backend 64k;\n        server backend:8080 resolve;\n    }' \
        /tmp/photo-album.conf > /etc/nginx/nginx.conf \
    && rm /tmp/photo-album.conf /etc/nginx/conf.d/default.conf
