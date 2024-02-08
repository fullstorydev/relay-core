from http.server import BaseHTTPRequestHandler, HTTPServer
import logging
import gzip
from io import BytesIO

class SimpleHTTPRequestHandler(BaseHTTPRequestHandler):

    def _set_response(self, content_type='text/html'):
        self.send_response(200)
        self.send_header('Content-type', content_type)
        self.end_headers()

    def _get_request_body(self):
        content_length = int(self.headers['Content-Length'])  # Get the size of the data
        request_data = self.rfile.read(content_length)  # Get the data itself

        # Check if the data is gzipped and decompress if necessary
        if self.headers.get('Content-Encoding') == 'gzip':
            request_data = gzip.decompress(request_data)

        return request_data.decode('utf-8')

    def do_GET(self):
        logging.info("GET request,\nPath: %s\nHeaders:\n%s\n", str(self.path), str(self.headers))
        self._set_response()
        self.wfile.write(f"GET request for {self.path}".encode('utf-8'))

    def do_POST(self):
        post_data = self._get_request_body()
        logging.info("POST request,\nPath: %s\nHeaders:\n%s\n\nBody:\n%s\n",
                     str(self.path), str(self.headers), post_data)

        self._set_response()
        self.wfile.write(f"POST request for {self.path}".encode('utf-8'))

    def do_PUT(self):
        put_data = self._get_request_body()
        logging.info("PUT request,\nPath: %s\nHeaders:\n%s\n\nBody:\n%s\n",
                     str(self.path), str(self.headers), put_data)

        self._set_response()
        self.wfile.write(f"PUT request for {self.path}".encode('utf-8'))

    def do_HEAD(self):
        logging.info("HEAD request,\nPath: %s\nHeaders:\n%s\n", str(self.path), str(self.headers))
        self._set_response()
        # Traditionally, HEAD requests do not have a body
        # Removed the line writing data to self.wfile since HEAD response should not include a body

def run(server_class=HTTPServer, handler_class=SimpleHTTPRequestHandler, port=8080):
    logging.basicConfig(level=logging.INFO)
    server_address = ('', port)
    httpd = server_class(server_address, handler_class)
    logging.info('Starting httpd...\n')
    try:
        httpd.serve_forever()
    except KeyboardInterrupt:
        pass
    httpd.server_close()
    logging.info('Stopping httpd...\n')

if __name__ == '__main__':
    from sys import argv

    if len(argv) == 2:
        run(port=int(argv[1]))
    else:
        run()