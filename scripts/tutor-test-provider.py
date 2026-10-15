#!/usr/bin/env python3
"""Local deterministic HTTP test double. No LLM, keys, storage, or external calls."""
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import json
class Provider(BaseHTTPRequestHandler):
    def log_message(self, *args): pass
    def do_POST(self):
        length = int(self.headers.get('Content-Length', '0'))
        if length > 65536:
            self.send_error(413); return
        data = json.loads(self.rfile.read(length))
        text = str(data.get('messages', []))
        if 'FAIL_PROVIDER' in text:
            self.send_response(503); self.end_headers(); return
        if 'Резервная' in text and data.get('model') == 'economy-test':
            self.send_response(429); self.end_headers(); return
        result = {'model': data.get('model'), 'choices': [{'message': {'content': 'Цикл повторяет команды. Проверь условие завершения.\n\n![Недоверенная картинка](https://tracker.invalid/tutor-pixel)'}}], 'usage': {'prompt_tokens': 20, 'completion_tokens': 10}}
        self.send_response(200); self.send_header('Content-Type','application/json'); self.end_headers(); self.wfile.write(json.dumps(result).encode())
ThreadingHTTPServer(('127.0.0.1', 8095), Provider).serve_forever()
