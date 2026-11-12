filepath = "/etc/nginx/sites-available/maicivy"

with open(filepath, "r") as f:
    content = f.read()

og_block = """
    # OG image - servie par Next.js (doit etre AVANT /api/)
    location /api/og {
        proxy_pass http://127.0.0.1:3002;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

"""

# Insert before "# Backend API"
if "location /api/og" in content:
    print("Block already present, skipping.")
elif "# Backend API" in content:
    content = content.replace("    # Backend API", og_block + "    # Backend API")
    with open(filepath, "w") as f:
        f.write(content)
    print("PATCHED OK")
else:
    print("ERROR: anchor not found")
    print("Content preview:")
    print(content[:2000])
