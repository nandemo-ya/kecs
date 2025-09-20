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

    // OG image generated successfully

    await browser.close();
})().catch(err => {
    process.stderr.write(`Error generating OG image: ${err.message}\n`);
    process.exit(1);
});
