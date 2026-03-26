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

## Subpath on an existing site (`https://example.com/tracelog/`)

Use this when another app already serves `/` on the same host (e.g. `cadourile.ro`).

1. Run the installer with a path prefix (no trailing slash). To **auto-edit** your SSL vhost (filename under `sites-enabled`, e.g. `cadourile.ro`):

   ```bash
   sudo TRACELOG_URL_PREFIX=/tracelog TRACELOG_NGINX_SITE=cadourile.ro bash scripts/install.sh
   ```

   Optional: **`TRACELOG_PUBLIC_DOMAIN=cadourile.ro`** if the banner should show a hostname different from the vhost filename.

   Or set **`Environment=TRACELOG_URL_PREFIX=/tracelog`** and **`--url-prefix /tracelog`** on `tracelog serve` in systemd.

2. The installer writes **`/etc/nginx/conf.d/tracelog-subpath-map.conf`** (WebSocket `map`) and **`/etc/nginx/snippets/tracelog-subpath-loc.conf`** (`location /tracelog/` → `http://127.0.0.1:8090/` with the path stripped).

3. If you did **not** pass **`TRACELOG_NGINX_SITE`**, add inside your **existing** `server { }` for HTTPS:

   ```nginx
   include /etc/nginx/snippets/tracelog-subpath-loc.conf;
   ```

   Then `sudo nginx -t && sudo systemctl reload nginx`.

4. Open **`https://your-domain/tracelog/`**. For **remote agents**, set the hub URL to **`https://your-domain/tracelog`** (no trailing slash is fine).

The UI and API use the prefix at runtime (embedded `index.html` placeholder + cookie `Path`).

## When Using a Reverse Proxy

Bind TraceLog to localhost only:

```bash
tracelog serve --bind 127.0.0.1 --port 8090 --url-prefix /tracelog
```

Omit `--url-prefix` when the proxy serves TraceLog at `/`. This prevents direct access, forcing all traffic through the proxy.
