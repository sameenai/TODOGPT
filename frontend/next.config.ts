import path from 'path';
import type { NextConfig } from 'next';

const goBackend = process.env.GO_BACKEND_URL || 'http://localhost:8080';

const nextConfig: NextConfig = {
  outputFileTracingRoot: path.join(__dirname, '..'),
  async rewrites() {
    return [
      {
        source: '/api/go/:path*',
        destination: `${goBackend}/api/:path*`,
      },
    ];
  },
};

export default nextConfig;
