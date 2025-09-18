#!/bin/bash
# Generate OG image for KECS documentation site

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Generating OG image for KECS documentation...${NC}"

# Check if required tools are installed
command -v npm >/dev/null 2>&1 || {
    echo -e "${RED}Error: npm is required but not installed.${NC}"
    exit 1
}

# Navigate to docs-site directory
cd "$(dirname "$0")/../docs-site"

# Install puppeteer if not already installed
if [ ! -d "node_modules/puppeteer" ]; then
    echo -e "${YELLOW}Installing puppeteer...${NC}"
    npm install --save-dev puppeteer
fi

# Create Node.js script to generate the image (using .cjs for CommonJS)
cat > generate-og.cjs << 'EOF'
const puppeteer = require('puppeteer');
const fs = require('fs');
const path = require('path');

(async () => {
    const browser = await puppeteer.launch({
        headless: true,
        args: ['--no-sandbox', '--disable-setuid-sandbox']
    });

    const page = await browser.newPage();

    // Set viewport to OG image size
    await page.setViewport({
        width: 1200,
        height: 630,
        deviceScaleFactor: 2 // For better quality
    });

    // Read the HTML template
    const htmlPath = path.join(__dirname, 'public', 'og-template.html');
    const htmlContent = fs.readFileSync(htmlPath, 'utf8');

    // Load the HTML content
    await page.setContent(htmlContent);

    // Wait for fonts to load
    await page.evaluateHandle('document.fonts.ready');

    // Take screenshot
    const outputPath = path.join(__dirname, 'public', 'og-image.png');
    await page.screenshot({
        path: outputPath,
        type: 'png'
    });

    console.log(`✅ OG image generated at: ${outputPath}`);

    await browser.close();
})().catch(err => {
    console.error('❌ Error generating OG image:', err);
    process.exit(1);
});
EOF

# Run the Node.js script
echo -e "${YELLOW}Generating image...${NC}"
node generate-og.cjs

# Clean up
rm generate-og.cjs

echo -e "${GREEN}✅ OG image generation complete!${NC}"
echo -e "${GREEN}Image saved to: docs-site/public/og-image.png${NC}"