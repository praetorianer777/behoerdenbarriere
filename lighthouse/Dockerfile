FROM node:24-slim

# Lighthouse steuert den Browser selbst, deshalb liegt Chromium im selben Image.
RUN apt-get update && apt-get install -y --no-install-recommends \
      chromium fonts-liberation fonts-dejavu-core ca-certificates \
    && rm -rf /var/lib/apt/lists/*

ENV CHROME_PATH=/usr/bin/chromium
WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm install --omit=dev
COPY server.mjs ./
EXPOSE 3000
CMD ["node", "server.mjs"]
