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
      description: 'DiffusionGemma as a Zero-Shot Decision Model & Declarative Policy-as-Template Engine across Apple Silicon, GCE, and Cloud Run',
      social: [
        { icon: 'github', label: 'GitHub', href: 'https://github.com/ghchinoy/dgem' },
      ],
      plugins: [
        catppuccin({
          light: { flavor: 'latte', accent: 'mauve' },
          dark: { flavor: 'mocha', accent: 'mauve' },
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
            { label: 'Decision Studio, MCP & HTTP API', slug: 'studio-mcp-api' },
            { label: 'Setup & Metal Engine', slug: 'setup' },
            { label: 'dgem User Guide', slug: 'user-guide' },
          ],
        },
        {
          label: 'Core Architecture',
          items: [
            { label: 'The Journey to Decision Models', slug: 'decision-models-primer' },
            { label: 'Discrete Diffusion vs. Autoregression', slug: 'architecture' },
            { label: 'Glossary & Mental Models', slug: 'glossary' },
            { label: 'Template Catalog (Policy-as-Code)', slug: 'templates' },
          ],
        },
        {
          label: 'Experiments & Research Log',
          items: [
            { label: 'Experiment Ledger (EXP-01 – EXP-10)', slug: 'experiments' },
            { label: 'Next-Horizon Cascades & Policy DAGs', slug: 'experiments/exp-05-roadmap-cascades-and-dags' },
            { label: 'Listwise Diffusion Reranking (EXP-10)', slug: 'experiments/exp-10-listwise-diffusion-reranking' },
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
