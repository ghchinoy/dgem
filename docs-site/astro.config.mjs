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
      node.value = `<div class="mermaid-wrapper" title="Click to expand diagram in full-screen lightbox"><button type="button" class="mermaid-zoom-btn" aria-label="Expand diagram">🔍 Click to Zoom</button><pre class="mermaid" data-mermaid-src="${ encodeURIComponent(node.value) }">${ escapeHtml(node.value) }</pre></div>`;
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

            let zoomScale = 1;
            let panX = 0;
            let panY = 0;
            let isDragging = false;
            let dragStartX = 0;
            let dragStartY = 0;

            function ensureLightboxModal() {
              let modal = document.getElementById('mermaid-lightbox-modal');
              if (modal) return modal;
              modal = document.createElement('div');
              modal.id = 'mermaid-lightbox-modal';
              modal.className = 'mermaid-lightbox';
              modal.innerHTML = \`
                <div class="mermaid-lightbox-toolbar">
                  <span class="mermaid-lightbox-title">🔍 Interactive Diagram View (Scroll to Zoom · Drag to Pan)</span>
                  <div class="mermaid-lightbox-controls">
                    <button type="button" id="mm-zoom-out" title="Zoom Out (-)">−</button>
                    <button type="button" id="mm-zoom-reset" title="Reset Zoom (100%)">100%</button>
                    <button type="button" id="mm-zoom-in" title="Zoom In (+)">+</button>
                    <button type="button" id="mm-close" class="mm-close-btn" title="Close (Esc)">✕ Close</button>
                  </div>
                </div>
                <div class="mermaid-lightbox-stage" id="mm-stage">
                  <div class="mermaid-lightbox-canvas" id="mm-canvas"></div>
                </div>
              \`;
              document.body.appendChild(modal);

              const canvas = modal.querySelector('#mm-canvas');
              const stage = modal.querySelector('#mm-stage');
              const resetBtn = modal.querySelector('#mm-zoom-reset');

              function applyTransform() {
                canvas.style.transform = \`translate(\${panX}px, \${panY}px) scale(\${zoomScale})\`;
                resetBtn.textContent = \`\${Math.round(zoomScale * 100)}%\`;
              }

              modal.querySelector('#mm-zoom-in').addEventListener('click', (e) => {
                e.stopPropagation();
                zoomScale = Math.min(4, +(zoomScale + 0.25).toFixed(2));
                applyTransform();
              });

              modal.querySelector('#mm-zoom-out').addEventListener('click', (e) => {
                e.stopPropagation();
                zoomScale = Math.max(0.4, +(zoomScale - 0.25).toFixed(2));
                applyTransform();
              });

              resetBtn.addEventListener('click', (e) => {
                e.stopPropagation();
                zoomScale = 1;
                panX = 0;
                panY = 0;
                applyTransform();
              });

              modal.querySelector('#mm-close').addEventListener('click', () => {
                modal.classList.remove('open');
              });

              modal.addEventListener('click', (e) => {
                if (e.target === modal || e.target === stage) {
                  modal.classList.remove('open');
                }
              });

              stage.addEventListener('wheel', (e) => {
                e.preventDefault();
                const delta = e.deltaY < 0 ? 0.15 : -0.15;
                zoomScale = Math.min(4, Math.max(0.4, +(zoomScale + delta).toFixed(2)));
                applyTransform();
              }, { passive: false });

              stage.addEventListener('mousedown', (e) => {
                isDragging = true;
                dragStartX = e.clientX - panX;
                dragStartY = e.clientY - panY;
                stage.style.cursor = 'grabbing';
              });

              window.addEventListener('mousemove', (e) => {
                if (!isDragging) return;
                panX = e.clientX - dragStartX;
                panY = e.clientY - dragStartY;
                applyTransform();
              });

              window.addEventListener('mouseup', () => {
                isDragging = false;
                if (stage) stage.style.cursor = 'grab';
              });

              window.addEventListener('keydown', (e) => {
                if (e.key === 'Escape' && modal.classList.contains('open')) {
                  modal.classList.remove('open');
                }
              });

              return modal;
            }

            function openMermaidLightbox(wrapper) {
              const svg = wrapper.querySelector('svg');
              if (!svg) return;
              const modal = ensureLightboxModal();
              const canvas = modal.querySelector('#mm-canvas');
              canvas.innerHTML = '';
              const clone = svg.cloneNode(true);
              clone.removeAttribute('style');
              clone.style.width = '100%';
              clone.style.height = 'auto';
              clone.style.maxHeight = '82vh';
              canvas.appendChild(clone);
              zoomScale = 1;
              panX = 0;
              panY = 0;
              canvas.style.transform = 'translate(0px, 0px) scale(1)';
              modal.querySelector('#mm-zoom-reset').textContent = '100%';
              modal.classList.add('open');
            }

            async function renderAllMermaid() {
              const isDark = document.documentElement.dataset.theme !== 'light';
              mermaid.initialize({
                startOnLoad: false,
                theme: isDark ? 'dark' : 'default',
                securityLevel: 'loose',
                fontFamily: 'Inter, system-ui, sans-serif',
                flowchart: { useMaxWidth: true, htmlLabels: true, curve: 'basis' },
                themeVariables: { fontSize: '15px' },
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
              document.querySelectorAll('.mermaid-wrapper').forEach((wrapper) => {
                if (!wrapper.dataset.lightboxBound) {
                  wrapper.dataset.lightboxBound = 'true';
                  wrapper.addEventListener('click', () => openMermaidLightbox(wrapper));
                }
              });
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
            { label: 'Experiment Authoring & Backend Guide', slug: 'experiment-authoring-guide' },
            { label: 'Custom Dataset & Experiment Cookbook', slug: 'custom-dataset-guide' },
            { label: 'CLI, HTTP Gateway & MCP Reference', slug: 'cli-reference' },
            { label: 'Setup & Metal Engine', slug: 'setup' },
            { label: 'dgem User Guide', slug: 'user-guide' },
          ],
        },
        {
          label: 'Core Architecture',
          items: [
            { label: 'The Journey to Decision Models', slug: 'decision-models-primer' },
            { label: 'Confidence Beyond Shannon (IDC)', slug: 'confidence-beyond-shannon' },
            { label: 'Discrete Diffusion vs. Autoregression', slug: 'architecture' },
            { label: 'Unclassified Grouping & Taxonomy Discovery', slug: 'taxonomy-discovery' },
            { label: 'Glossary & Mental Models', slug: 'glossary' },
            { label: 'Template Catalog (Policy-as-Code)', slug: 'templates' },
          ],
        },
        {
          label: 'Experiments & Research Log',
          items: [
            { label: 'Experiment Ledger (EXP-01 – EXP-13)', slug: 'experiments' },
            { label: 'Proposed Experiments Register', slug: 'experiments/proposed' },
            { label: 'Next-Horizon Cascades & Policy DAGs', slug: 'experiments/exp-05-roadmap-cascades-and-dags' },
            { label: 'Listwise Diffusion Reranking (EXP-10)', slug: 'experiments/exp-10-listwise-diffusion-reranking' },
            { label: 'JevBench v1.3.1 Parity & Sync (EXP-11)', slug: 'experiments/exp-11-jevbench-parity' },
            { label: 'Decision Index & Wide-Canvas (EXP-12)', slug: 'experiments/exp-12-decision-index' },
            { label: 'Permutation & Dual-Mirror Canvas (EXP-13)', slug: 'experiments/exp-13-permutation-invariance' },
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
            { label: 'Vertex AI (/invoke/*) vs. Cloud Run', slug: 'vertex-ai-vs-cloudrun' },
            { label: 'Cloud Run Architecture & Lessons', slug: 'cloudrun-lessons-learned' },
            { label: 'OTel Traces & Cloud Observability', slug: 'observability-traces' },
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
