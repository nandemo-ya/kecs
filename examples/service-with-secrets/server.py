import os
import json
from http.server import HTTPServer, BaseHTTPRequestHandler

class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == '/health':
            self.send_response(200)
            self.send_header('Content-Type', 'application/json')
            self.end_headers()
            self.wfile.write(json.dumps({'status': 'healthy'}).encode())
        elif self.path == '/config':
            self.send_response(200)
            self.send_header('Content-Type', 'application/json')
            self.end_headers()
            config = {
                'database_url': os.environ.get('DATABASE_URL', 'not_set'),
                'api_key_present': bool(os.environ.get('API_KEY')),
                'app_config': os.environ.get('APP_CONFIG', 'not_set'),
                'feature_flags': os.environ.get('FEATURE_FLAGS', 'not_set'),
                'environment': os.environ.get('ENVIRONMENT', 'not_set')
            }
            self.wfile.write(json.dumps(config, indent=2).encode())
        elif self.path == '/secrets':
            self.send_response(200)
            self.send_header('Content-Type', 'application/json')
            self.end_headers()
            # Never expose actual secrets, just confirm they exist
            secrets = {
                'db_password_loaded': bool(os.environ.get('DB_PASSWORD')),
                'jwt_secret_loaded': bool(os.environ.get('JWT_SECRET')),
                'encryption_key_loaded': bool(os.environ.get('ENCRYPTION_KEY'))
            }
            self.wfile.write(json.dumps(secrets, indent=2).encode())
        else:
            self.send_response(404)
            self.end_headers()

if __name__ == '__main__':
    print('Starting server on port 8080...')
    print('Environment:', os.environ.get('ENVIRONMENT', 'not_set'))
    httpd = HTTPServer(('', 8080), Handler)
    httpd.serve_forever()