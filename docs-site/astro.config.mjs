import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';
import catppuccin from '@catppuccin/starlight';
import remarkMath from 'remark-math';
import rehypeKatex from 'rehype-katex';

export default defineConfig({
  site: 'https://ghchinoy.github.io',
  base: '/dgem',
  markdown: {
    remarkPlugins: [remarkMath],
    rehypePlugins: [rehypeKatex],
  },
  integrations: [
    starlight({
      title: 'DiffusionGemma',
      description: 'Discrete Diffusion Slot Readouts, Multi-Environment Orchestration & Benchmarks across Apple Silicon, GCE, and Cloud Run',
      social: [
        { icon: 'github', label: 'GitHub', href: 'https://github.com/ghchinoy/dgem' },
      ],
      plugins: [
        catppuccin({
          light: { flavor: 'latte', accent: 'mauve' },
          dark: { flavor: 'latte', accent: 'mauve' },
        }),
      ],
      customCss: [
        'katex/dist/katex.min.css',
        './src/styles/custom.css',
      ],
      sidebar: [
        {
          label: 'Getting Started',
          items: [
            { label: 'Overview', link: '/' },
            { label: 'Setup & Metal Engine', slug: 'setup' },
            { label: 'dgem User Guide', slug: 'user-guide' },
          ],
        },
        {
          label: 'Core Architecture',
          items: [
            { label: 'The Journey to Decision Models', slug: 'decision-models-primer' },
            { label: 'Discrete Diffusion vs. Autoregression', slug: 'architecture' },
            { label: 'Template Catalog', slug: 'templates' },
          ],
        },
        {
          label: 'Use Cases & Patterns',
          items: [
            { label: 'Real-World Applications', slug: 'applications' },
          ],
        },
        {
          label: 'Deployment & Cloud',
          items: [
            { label: 'Remote Endpoints & Cloud', slug: 'remote-endpoints' },
            { label: 'Cloud Run Architecture & Lessons', slug: 'cloudrun-lessons-learned' },
          ],
        },
        {
          label: 'Benchmarks & Evaluation',
          items: [
            { label: 'Evaluation Report (Metal vs. Cloud)', slug: 'benchmarks' },
            { label: 'Ecotone (WFST) vs. DiffusionGemma', slug: 'ecotone-comparison' },
          ],
        },
      ],
    }),
  ],
});
