#!/usr/bin/env node
/**
 * Mock HTTP server for testing agent-metadata-action
 *
 * This server mocks the NewRelic metadata service endpoint.
 * It accepts any POST request and returns a 200 OK with a success message.
 *
 * Usage: node mock-server.js <port>
 */

const http = require('http');

const port = process.argv[2] || 8080;

const server = http.createServer((req, res) => {
  const timestamp = new Date().toISOString();
  console.log(`[${timestamp}] ${req.method} ${req.url}`);
  console.log(`Headers:`, JSON.stringify(req.headers, null, 2));

  // Log request body for POST/PUT requests
  if (req.method === 'POST' || req.method === 'PUT') {
    let body = '';
    req.on('data', chunk => {
      body += chunk.toString();
    });
    req.on('end', () => {
      console.log(`Body length: ${body.length} bytes`);
      if (body.length < 1000) {
        console.log(`Body preview:`, body.substring(0, 500));
      }

      // Send success response
      res.writeHead(200, {
        'Content-Type': 'application/json',
        'X-Mock-Server': 'true'
      });
      res.end(JSON.stringify({
        status: 'success',
        message: 'Mock metadata service accepted request',
        timestamp: timestamp,
        receivedBytes: body.length
      }));
    });
  } else {
    // For GET/other methods, just return success
    res.writeHead(200, {
      'Content-Type': 'application/json',
      'X-Mock-Server': 'true'
    });
    res.end(JSON.stringify({
      status: 'success',
      message: 'Mock metadata service is running',
      timestamp: timestamp
    }));
  }
});

server.listen(port, '127.0.0.1', () => {
  console.log(`Mock metadata service listening on http://127.0.0.1:${port}`);
  console.log(`Ready to accept requests...`);
});

// Handle graceful shutdown
process.on('SIGTERM', () => {
  console.log('Received SIGTERM, shutting down gracefully...');
  server.close(() => {
    console.log('Server closed');
    process.exit(0);
  });
});

process.on('SIGINT', () => {
  console.log('Received SIGINT, shutting down gracefully...');
  server.close(() => {
    console.log('Server closed');
    process.exit(0);
  });
});
