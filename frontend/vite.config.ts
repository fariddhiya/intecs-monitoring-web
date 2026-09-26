import { sveltekit } from '@sveltejs/kit/vite';
import staticAdapter from '@sveltejs/adapter-static';

const config = {
  plugins: [sveltekit({
    adapter: staticAdapter({
      pages: 'dist',
      assets: 'dist',
      fallback: 'index.html',
    }),
  })],
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        ws: true,
      },
    },
  },
};

export default config;
