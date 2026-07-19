FROM nginx:alpine

RUN mkdir -p /etc/nginx/ssl && \
    openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
        -keyout /etc/nginx/ssl/private.key \
        -out /etc/nginx/ssl/certificate.crt \
        -subj "/CN=localhost/O=LidarPlatform/C=RU"

COPY nginx.conf /etc/nginx/nginx.conf
