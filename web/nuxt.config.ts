import tailwindcss from '@tailwindcss/vite'

// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  modules: [
    '@nuxt/eslint',
    '@nuxt/icon',
    '@vueuse/nuxt',
    'motion-v/nuxt',
    'nuxt-api-party',
  ],
  devtools: { enabled: true },

  app: {
    head: {
      title: 'Mochi',
      htmlAttrs: { lang: 'en' },
      viewport: 'width=device-width, initial-scale=1.0',
      charset: 'utf-8',
      link: [
        {
          rel: 'preconnect',
          href: 'https://fonts.bunny.net',
        },
        {
          rel: 'stylesheet',
          href: 'https://fonts.bunny.net/css?family=inconsolata:400,500,600,700|karla:400,500,600,700',
        },
      ],
      script: [
        {
          // Prevent FOUC
          innerHTML: `
            (function() {
              const theme = localStorage.getItem('mochi-theme');
              const isDark = theme === 'dark' || (!theme && window.matchMedia('(prefers-color-scheme: dark)').matches);
              if (isDark) {
                document.documentElement.classList.add('dark');
              } else {
                document.documentElement.classList.remove('dark');
              }
            })();
          `,
          type: 'text/javascript',
        },
      ],
    },
  },

  css: ['~/assets/css/main.css'],

  runtimeConfig: {
    apiParty: {
      endpoints: {
        api: {
          url: 'http://localhost:8080',
        },
      },
    },
  },
  compatibilityDate: '2025-07-15',
  vite: {
    plugins: [tailwindcss()],
    optimizeDeps: {
      include: [
        '@floating-ui/vue',
        'echarts/charts',
        'echarts/components',
        'echarts/core',
        'echarts/renderers',
      ],
    },
  },

  eslint: {
    config: {
      stylistic: true,
    },
  },

  icon: {
    mode: 'css',
    cssLayer: 'base',
  },
})
