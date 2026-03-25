# Reverse Proxy

For production use, put TraceLog behind a reverse proxy with HTTPS.

## nginx

```nginx
server {
    listen 443 ssl http2;
    server_name monitor.example.com;

    ssl_certificate /etc/letsencrypt/live/monitor.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/monitor.example.com/privkey.pem;

    location / {
        proxy_pass http://127.0.0.1:8090;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # WebSocket support for agent connections
    location /api/ws/ {
        proxy_pass http://127.0.0.1:8090;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}

# Redirect HTTP to HTTPS
server {
    listen 80;
    server_name monitor.example.com;
    return 301 https://$host$request_uri;
}
```

## Caddy

```caddyfile
monitor.example.com {
    reverse_proxy 127.0.0.1:8090
}
```

Caddy automatically handles HTTPS certificates.

## When Using a Reverse Proxy

Bind TraceLog to localhost only:

```bash
tracelog serve --bind 127.0.0.1 --port 8090
```

This prevents direct access, forcing all traffic through the proxy.
