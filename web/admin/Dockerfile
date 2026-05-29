FROM node:22-alpine AS builder

WORKDIR /app

COPY package*.json ./
RUN npm ci

COPY . ./

ARG VITE_API_BASE_URL=http://localhost:18080/v1
ARG VITE_JWT_SECRET=fireshine
ENV VITE_API_BASE_URL=${VITE_API_BASE_URL}
ENV VITE_JWT_SECRET=${VITE_JWT_SECRET}

RUN npm run build

FROM nginx:1.27-alpine

COPY deploy/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/dist /usr/share/nginx/html

EXPOSE 80
