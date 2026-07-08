#!/usr/bin/env python3
"""Minimal org-key service for locally testing private ingredients.

Serves the organization encryption key over HTTPS at GET /v1/org-key in the
contract the State Tool expects. For local testing only; not a shipped artifact.

Generate a self-signed certificate (the SAN must cover the host you connect to):

    openssl req -x509 -newkey rsa:2048 -nodes -days 365 \
        -keyout key.pem -out cert.pem \
        -subj "/CN=localhost" \
        -addext "subjectAltName=DNS:localhost,IP:127.0.0.1"

Run:

    python3 scripts/orgkeyserver.py --tls-cert cert.pem --tls-key key.pem

Point the State Tool at it (the base URL only; the tool appends /v1/org-key):

    state config set privateingredient.key_service_url https://127.0.0.1:8443
    state config set privateingredient.key_service_ca   /path/to/cert.pem
    # Optional bearer auth (start the server with --token <token>):
    state config set privateingredient.bearer_token_env ORGKEY_TOKEN
    export ORGKEY_TOKEN=<token>

`state publish --build` and the later install must share the same key, so reuse
the printed --key value across runs.
"""

import argparse
import base64
import hashlib
import json
import secrets
import ssl
import sys
from http.server import BaseHTTPRequestHandler, HTTPServer

KEY_SIZE = 32  # AES-256
ENDPOINT = "/v1/org-key"


def build_contract(org, key_id, raw_key):
    return {
        "schema": "activestate.pim.orgkey/v1",
        "org": org,
        "key_id": key_id,
        "algorithm": "AES-256-GCM",
        "encoding": "base64",
        "key": "b64:" + base64.standard_b64encode(raw_key).decode("ascii"),
        "fingerprint": "sha256:" + hashlib.sha256(raw_key).hexdigest(),
    }


def make_handler(contract, token):
    body = json.dumps(contract).encode("utf-8")

    class Handler(BaseHTTPRequestHandler):
        def do_GET(self):
            if self.path != ENDPOINT:
                self.send_error(404)
                return
            if token and self.headers.get("Authorization") != "Bearer " + token:
                self.send_error(401)
                return
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)

    return Handler


def parse_key(encoded):
    raw = base64.standard_b64decode(encoded)
    if len(raw) != KEY_SIZE:
        raise SystemExit(f"--key must decode to {KEY_SIZE} bytes, got {len(raw)}")
    return raw


def main():
    p = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    p.add_argument("--tls-cert", required=True, help="server TLS certificate (PEM)")
    p.add_argument("--tls-key", required=True, help="server TLS private key (PEM)")
    p.add_argument("--org", default="ActiveState-CLI-Testing",
                   help="organization the key belongs to; must match the project owner")
    p.add_argument("--key", help="base64-encoded 32-byte AES key; generated and printed if omitted")
    p.add_argument("--key-id", default="orgkey-test", help="opaque key identifier")
    p.add_argument("--host", default="127.0.0.1")
    p.add_argument("--port", type=int, default=8443)
    p.add_argument("--token", help="if set, require this value as a bearer token")
    args = p.parse_args()

    if args.key:
        raw_key = parse_key(args.key)
    else:
        raw_key = secrets.token_bytes(KEY_SIZE)
        print("--key", base64.standard_b64encode(raw_key).decode("ascii"), file=sys.stderr)

    ctx = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
    ctx.minimum_version = ssl.TLSVersion.TLSv1_2
    ctx.load_cert_chain(certfile=args.tls_cert, keyfile=args.tls_key)

    handler = make_handler(build_contract(args.org, args.key_id, raw_key), args.token)
    httpd = HTTPServer((args.host, args.port), handler)
    httpd.socket = ctx.wrap_socket(httpd.socket, server_side=True)
    print(f"serving {ENDPOINT} for org {args.org!r} on https://{args.host}:{args.port}", file=sys.stderr)
    try:
        httpd.serve_forever()
    except KeyboardInterrupt:
        httpd.shutdown()


if __name__ == "__main__":
    main()
