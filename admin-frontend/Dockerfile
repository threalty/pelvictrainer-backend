# Stage 1: Build React app
FROM node:20-alpine AS builder

WORKDIR /app

COPY package.json package-lock.json* ./
RUN npm ci --legacy-peer-deps

COPY . .
RUN npm run build

# Stage 2: Serve via Nginx
FROM nginx:alpine

COPY nginx.conf /etc/nginx/conf.d/default.conf

# ИЗМЕНЕНО: копируем в подпапку /admin
COPY --from=builder /app/dist /usr/share/nginx/html/admin

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]