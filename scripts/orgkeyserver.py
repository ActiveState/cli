#!/usr/bin/env python3
"""Minimal org-key service for locally testing private ingredients.

Serves the organization encryption key over HTTPS at GET /v1/org-key in the
contract the State Tool expects. For local testing only; not a shipped artifact.

Generate a self-signed certificate (the SAN must cover the host you connect to):

    openssl req -x509 -newkey rsa:2048 -nodes -days 365 \
        -keyout key.pem -out cert.pem \
        -subj "/CN=localhost" \
        -addext "subjectAltName=DNS:localhost,IP:127.0.0.1"

Or let this script generate one for you (needs the 'cryptography' package, no
openssl binary required -- useful on native Windows / cmd):

    python3 scripts/orgkeyserver.py --gen-cert --tls-cert cert.pem --tls-key key.pem

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
import os
import secrets
import ssl
import sys
from http.server import BaseHTTPRequestHandler, HTTPServer

KEY_SIZE = 32  # AES-256
ENDPOINT = "/v1/org-key"


def generate_self_signed(cert_path, key_path, host):
    """Write a self-signed cert/key pair to the given paths.

    Uses the 'cryptography' package so no external openssl binary is required
    (openssl is frequently unavailable on native Windows / cmd environments).
    """
    try:
        import datetime
        import ipaddress

        from cryptography import x509
        from cryptography.hazmat.primitives import hashes, serialization
        from cryptography.hazmat.primitives.asymmetric import rsa
        from cryptography.x509.oid import NameOID
    except ImportError:
        raise SystemExit(
            "--gen-cert requires the 'cryptography' package (pip install "
            "cryptography), or supply --tls-cert/--tls-key generated with openssl."
        )

    key = rsa.generate_private_key(public_exponent=65537, key_size=2048)
    name = x509.Name([x509.NameAttribute(NameOID.COMMON_NAME, "localhost")])
    dns_names = ["localhost"]
    ip_addrs = ["127.0.0.1"]
    try:
        ipaddress.ip_address(host)
        if host not in ip_addrs:
            ip_addrs.append(host)
    except ValueError:
        if host not in dns_names:
            dns_names.append(host)
    alt_names = [x509.DNSName(n) for n in dns_names]
    alt_names += [x509.IPAddress(ipaddress.ip_address(ip)) for ip in ip_addrs]
    now = datetime.datetime.utcnow()
    cert = (
        x509.CertificateBuilder()
        .subject_name(name)
        .issuer_name(name)
        .public_key(key.public_key())
        .serial_number(x509.random_serial_number())
        .not_valid_before(now - datetime.timedelta(minutes=1))
        .not_valid_after(now + datetime.timedelta(days=365))
        .add_extension(x509.SubjectAlternativeName(alt_names), critical=False)
        .sign(key, hashes.SHA256())
    )

    cert_dir = os.path.dirname(cert_path)
    if cert_dir:
        os.makedirs(cert_dir, exist_ok=True)
    key_dir = os.path.dirname(key_path)
    if key_dir:
        os.makedirs(key_dir, exist_ok=True)
    with open(key_path, "wb") as f:
        f.write(key.private_bytes(
            serialization.Encoding.PEM,
            serialization.PrivateFormat.TraditionalOpenSSL,
            serialization.NoEncryption(),
        ))
    # Restrict the private key to owner-only; no-op-ish on Windows.
    try:
        os.chmod(key_path, 0o600)
    except OSError:
        pass
    with open(cert_path, "wb") as f:
        f.write(cert.public_bytes(serialization.Encoding.PEM))


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
    p.add_argument("--gen-cert", action="store_true",
                   help="generate a self-signed cert/key at --tls-cert/--tls-key "
                        "before serving (no openssl binary required)")
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

    if args.gen_cert:
        generate_self_signed(args.tls_cert, args.tls_key, args.host)

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
