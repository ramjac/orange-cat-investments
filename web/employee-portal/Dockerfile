# Multi-stage build for Orange Cat Investments - Employee Web Portal
FROM node:20-alpine AS builder

WORKDIR /app

# Copy design tokens and theme generator
COPY config/branding.yaml config/branding.yaml
COPY scripts/generate-theme.js scripts/generate-theme.js

# Copy employee portal package definitions
COPY web/employee-portal/package*.json web/employee-portal/
WORKDIR /app/web/employee-portal
RUN npm ci

# Copy employee portal source tree and compile theme & production bundle
COPY web/employee-portal/ .
RUN npm run build

# Production runtime using minimal NGINX
FROM nginx:1.27-alpine

COPY web/employee-portal/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/web/employee-portal/dist /usr/share/nginx/html

EXPOSE 5173

STOPSIGNAL SIGQUIT

CMD ["nginx", "-g", "daemon off;"]
