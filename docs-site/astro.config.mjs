import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';
import catppuccin from '@catppuccin/starlight';
import remarkMath from 'remark-math';
import rehypeKatex from 'rehype-katex';

function remarkMermaid() {
  function escapeHtml(str) {
    return str
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;');
  }
  function walk(node) {
    if (!node) return;
    if (node.type === 'code' && node.lang === 'mermaid') {
      node.type = 'html';
      node.value = `<div class="mermaid-wrapper"><pre class="mermaid" data-mermaid-src="${ encodeURIComponent(node.value) }">${ escapeHtml(node.value) }</pre></div>`;
      delete node.lang;
      delete node.meta;
      return;
    }
    if (Array.isArray(node.children)) {
      for (const child of node.children) walk(child);
    }
  }
  return (tree) => walk(tree);
}

export default defineConfig({
  site: 'https://ghchinoy.github.io',
  base: '/dgem',
  markdown: {
    remarkPlugins: [remarkMermaid, remarkMath],
    rehypePlugins: [rehypeKatex],
  },
  integrations: [
    starlight({
      title: 'DiffusionGemma',
      description: 'DiffusionGemma as a Zero-Shot Decision Model & Declarative Policy-as-Template Engine across Apple Silicon, GCE, and Cloud Run',
      social: [
        { icon: 'github', label: 'GitHub', href: 'https://github.com/ghchinoy/dgem' },
      ],
      head: [
        {
          tag: 'script',
          attrs: { type: 'module' },
          content: `
            import mermaid from 'https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.esm.min.mjs';
            async function renderAllMermaid() {
              const isDark = document.documentElement.dataset.theme !== 'light';
              mermaid.initialize({
                startOnLoad: false,
                theme: isDark ? 'dark' : 'default',
                securityLevel: 'loose',
                fontFamily: 'Inter, system-ui, sans-serif',
              });
              const nodes = document.querySelectorAll('pre.mermaid');
              for (const el of nodes) {
                const raw = el.getAttribute('data-mermaid-src');
                if (raw) {
                  el.removeAttribute('data-processed');
                  el.textContent = decodeURIComponent(raw);
                }
              }
              if (nodes.length > 0) {
                await mermaid.run({ nodes });
              }
            }
            document.addEventListener('DOMContentLoaded', renderAllMermaid);
            document.addEventListener('astro:page-load', renderAllMermaid);
            const obs = new MutationObserver(() => renderAllMermaid());
            obs.observe(document.documentElement, { attributes: true, attributeFilter: ['data-theme'] });
          `,
        },
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
            { label: 'Experiment Ledger (EXP-01 – EXP-11)', slug: 'experiments' },
            { label: 'Next-Horizon Cascades & Policy DAGs', slug: 'experiments/exp-05-roadmap-cascades-and-dags' },
            { label: 'Listwise Diffusion Reranking (EXP-10)', slug: 'experiments/exp-10-listwise-diffusion-reranking' },
            { label: 'JevBench v1.3.1 Parity & Sync (EXP-11)', slug: 'experiments/exp-11-jevbench-parity' },
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
