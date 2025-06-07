# KECS Documentation Site

This directory contains the VitePress-based documentation site for KECS.

## Development

```bash
# Install dependencies
npm install

# Start development server
npm run docs:dev

# Build for production
npm run docs:build

# Preview production build
npm run docs:preview
```

## Structure

- `.vitepress/` - VitePress configuration and theme
- `api/` - API reference documentation
- `guides/` - User guides and tutorials
- `deployment/` - Deployment documentation
- `development/` - Developer documentation
- `architecture/` - Architecture documentation

## Building

Use the build script from the project root:

```bash
./scripts/build-docs.sh
```

## Deployment

The documentation site can be deployed to:
- GitHub Pages
- Netlify
- Vercel
- Any static hosting service

The built files are in `.vitepress/dist/`.