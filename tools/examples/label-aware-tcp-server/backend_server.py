#!/usr/bin/env python3
"""Tiny HTTP backend that identifies itself for label-aware tcp-server manual tests."""

from __future__ import annotations

import argparse
from http.server import BaseHTTPRequestHandler, HTTPServer


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--port", type=int, required=True)
    parser.add_argument("--body", required=True)
    args = parser.parse_args()

    body = args.body.encode()

    class Handler(BaseHTTPRequestHandler):
        def do_GET(self) -> None:
            self.send_response(200)
            self.send_header("Content-Type", "text/plain")
            self.end_headers()
            self.wfile.write(body)

        def log_message(self, fmt: str, *log_args) -> None:
            return

    server = HTTPServer(("127.0.0.1", args.port), Handler)
    print(f"backend listening :{args.port} body={args.body!r}", flush=True)
    server.serve_forever()


if __name__ == "__main__":
    main()
