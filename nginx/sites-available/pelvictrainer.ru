server {
    listen 80;
    listen [::]:80;
    server_name pelvictrainer.ru www.pelvictrainer.ru;

    location /.well-known/acme-challenge/ {
        alias /var/www/certbot/;
        allow all;
    }

    location / {
        return 301 https://$host$request_uri;
    }
}

server {
    listen 443 ssl http2;
    listen [::]:443 ssl http2;
    server_name pelvictrainer.ru www.pelvictrainer.ru;

    ssl_certificate /etc/letsencrypt/live/pelvictrainer.ru/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/pelvictrainer.ru/privkey.pem;
    include /etc/letsencrypt/options-ssl-nginx.conf;
    ssl_dhparam /etc/letsencrypt/ssl-dhparams.pem;

    location /.well-known/acme-challenge/ {
        alias /var/www/certbot/;
    }

    location / {
        return 200 "PelvicTrainer - Coming Soon";
        add_header Content-Type text/plain;
    }
}
