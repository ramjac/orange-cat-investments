# Multi-stage build for Orange Cat Investments - Customer Web Portal
FROM node:20-alpine AS builder

WORKDIR /app

# Copy design tokens and theme generator
COPY config/branding.yaml config/branding.yaml
COPY scripts/generate-theme.js scripts/generate-theme.js

# Copy customer portal package definitions
COPY web/customer-portal/package*.json web/customer-portal/
WORKDIR /app/web/customer-portal
RUN npm ci

# Copy customer portal source tree and compile theme & production bundle
COPY web/customer-portal/ .
RUN npm run build

# Production runtime using minimal NGINX
FROM nginx:1.27-alpine

COPY web/customer-portal/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/web/customer-portal/dist /usr/share/nginx/html

EXPOSE 5174

STOPSIGNAL SIGQUIT

CMD ["nginx", "-g", "daemon off;"]
