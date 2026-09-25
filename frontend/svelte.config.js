import adapter from '@sveltejs/adapter-static';

/** @type {import('@sveltejs/kit').config} */
const config = {
  kit: {
    adapter: adapter({
      pages: 'dist',
      assets: 'dist',
      fallback: null,
    }),
  },
};

export default config;
