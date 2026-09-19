import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';
import catppuccin from '@catppuccin/starlight';

export default defineConfig({
  site: 'https://ghchinoy.github.io',
  base: '/diffusiongemma',
  integrations: [
    starlight({
      title: 'DiffusionGemma',
      description: 'Single-Pass Discrete Block Diffusion & Slot Readout on Apple Silicon',
      social: [
        { icon: 'github', label: 'GitHub', href: 'https://github.com/ghchinoy/diffusiongemma' },
      ],
      plugins: [
        catppuccin({
          light: { flavor: 'latte', accent: 'mauve' },
          dark: { flavor: 'latte', accent: 'mauve' },
        }),
      ],
      customCss: ['./src/styles/custom.css'],
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
            { label: 'Discrete Diffusion vs. Autoregression', slug: 'architecture' },
            { label: 'Template Catalog', slug: 'templates' },
          ],
        },
        {
          label: 'Deployment & Cloud',
          items: [
            { label: 'Remote Endpoints & Cloud', slug: 'remote-endpoints' },
          ],
        },
      ],
    }),
  ],
});
