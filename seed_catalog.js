/**
 * GoAnsuran Catalog Seeder
 *
 * Fetches LIVE product data from api.ansuran2u.com and seeds it
 * into ChromaDB as knowledge documents with real deposit + monthly pricing.
 *
 * Run: node seed_catalog.js
 *
 * Data source: https://api.ansuran2u.com/v1/product/go-flexi?page={1-3}&per-page=40
 */

const BACKEND_URL = 'http://localhost:3000';
const CATALOG_API = 'https://api.ansuran2u.com/v1/product/go-flexi';
const TOTAL_PAGES = 3;
const PER_PAGE = 40;

// ─── Helpers ──────────────────────────────────────────────

function cleanTitle(title) {
  // Normalise: "Apple iPhone 17 Pro Max 1TB" → "iPhone 17 Pro Max 1TB"
  return title
    .replace(/^(Apple|Samsung|Google|Honor|HONOR|Oppo|OPPO)\s+/i, '')
    .replace(/\s+/g, ' ')
    .trim();
}

function extractStorage(title) {
  const m = title.match(/(\d+)\s*(TB|GB)/i);
  return m ? `${m[1]}${m[2].toUpperCase()}` : 'N/A';
}

function extractSeries(title) {
  // iPhone 17 Pro Max 1TB → "iPhone 17 Pro Max"
  const cleaned = cleanTitle(title);
  const m = cleaned.match(/^(iPhone\s+\d+(?:\s+(?:Pro|Pro Max|Plus|Mini|e))?)/i);
  if (m) return m[1];
  // Galaxy S25 Ultra → "Galaxy S25 Ultra"
  const g = cleaned.match(/^(Galaxy\s+\w+\s+\w+)/i);
  if (g) return g[1];
  // Pixel 10 Pro → "Pixel 10 Pro"
  const p = cleaned.match(/^(Pixel\s+\d+\w?)/i);
  if (p) return p[1];
  // iPad variants
  const ip = cleaned.match(/^(iPad\s+[\w\s]+?)(?:\s+\d+)/i);
  if (ip) return ip[1];
  return cleaned.split(/\s+/).slice(0, 3).join(' ');
}

// ─── Fetch all product pages ─────────────────────────────

async function fetchAllProducts() {
  const all = [];
  for (let page = 1; page <= TOTAL_PAGES; page++) {
    const url = `${CATALOG_API}?page=${page}&per-page=${PER_PAGE}`;
    console.log(`   📡 Fetching page ${page}/${TOTAL_PAGES}: ${url}`);
    try {
      const res = await fetch(url, {
        headers: {
          'Accept': 'application/json',
          'User-Agent': 'GoAnsuran-RAG-Seeder/1.0',
        },
      });
      if (!res.ok) {
        console.warn(`   ⚠️  Page ${page} returned HTTP ${res.status}, skipping.`);
        continue;
      }
      const json = await res.json();
      const items = json.items || json.data || [];
      console.log(`   ✅ Page ${page}: ${items.length} products fetched.`);
      all.push(...items);
    } catch (err) {
      console.warn(`   ⚠️  Page ${page} fetch error: ${err.message}`);
    }
    // Small delay to be polite
    if (page < TOTAL_PAGES) await new Promise(r => setTimeout(r, 500));
  }
  return all;
}

// ─── Group & format products into KB documents ───────────

function buildCatalogDocs(products) {
  // Group by brand
  const byBrand = {};
  for (const p of products) {
    const brand = (p.brand?.name || 'Other').trim();
    if (!byBrand[brand]) byBrand[brand] = [];
    byBrand[brand].push(p);
  }

  const docs = [];

  for (const [brand, items] of Object.entries(byBrand)) {
    // Sort: by series then storage
    items.sort((a, b) => cleanTitle(a.title).localeCompare(cleanTitle(b.title)));

    // For Apple, split into iPhone vs iPad sub-groups (they're big)
    if (brand === 'Apple') {
      const iphones = items.filter(p => /iphone/i.test(p.title));
      const ipads = items.filter(p => /ipad/i.test(p.title));
      const others = items.filter(p => !/iphone|ipad/i.test(p.title));

      if (iphones.length > 0) {
        docs.push(buildBrandDoc('Apple iPhone', iphones));
      }
      if (ipads.length > 0) {
        docs.push(buildBrandDoc('Apple iPad', ipads));
      }
      if (others.length > 0) {
        docs.push(buildBrandDoc('Apple Other', others));
      }
    } else {
      docs.push(buildBrandDoc(brand, items));
    }
  }

  docs.push(buildSummaryDoc(products));

  return docs;
}

function buildBrandDoc(brandLabel, items) {
  const lines = [];

  // Header line
  lines.push(`CATALOG: ${brandLabel} — GoFlexi Plan Pricing (Deposit + Monthly Installment)`);
  lines.push(`Total models available: ${items.length}`);
  lines.push('');

  // Table header
  lines.push('Model & Storage | Deposit (Upfront) | Monthly Installment');
  lines.push('--- | --- | ---');

  for (const item of items) {
    const title = cleanTitle(item.title);
    const deposit = item.deposit?.text || 'N/A';
    const monthly = item.amount?.text || 'N/A';
    lines.push(`${title} | ${deposit} | ${monthly}`);
  }

  lines.push('');
  lines.push('NOTE: Prices shown are for the GoFlexi (Rent-To-Own) plan with 36-month tenure.');
  lines.push('Deposit is the upfront payment. Monthly is the per-month installment.');
  lines.push('Prices may differ for other plans (GoAngkasa, JCL, BNPL).');

  return {
    title: `Catalog: ${brandLabel} Pricing (GoFlexi)`,
    content: lines.join('\n'),
  };
}

function parseRM(text) {
  if (!text) return Infinity;
  const m = text.replace(/,/g, '').match(/[\d.]+/);
  return m ? parseFloat(m[0]) : Infinity;
}

function buildSummaryDoc(products) {
  const byBrand = {};
  for (const p of products) {
    const brand = (p.brand?.name || 'Other').trim();
    if (!byBrand[brand]) byBrand[brand] = [];
    byBrand[brand].push(p);
  }

  const lines = [];
  lines.push('CATALOG SUMMARY: Cheapest and Most Expensive Models Per Brand (GoFlexi Plan)');
  lines.push(`Total products in catalog: ${products.length}`);
  lines.push('');

  for (const [brand, items] of Object.entries(byBrand)) {
    let sorted = [...items].sort((a, b) => parseRM(a.deposit?.text) - parseRM(b.deposit?.text));
    const cheapest = sorted[0];
    const expensive = sorted[sorted.length - 1];

    lines.push(`${brand} (${sorted.length} models):`);
    lines.push(`  Lowest deposit: ${cleanTitle(cheapest.title)} — ${cheapest.deposit?.text || 'N/A'} deposit, ${cheapest.amount?.text || 'N/A'}/month`);
    lines.push(`  Highest deposit: ${cleanTitle(expensive.title)} — ${expensive.deposit?.text || 'N/A'} deposit, ${expensive.amount?.text || 'N/A'}/month`);
    lines.push('');
  }

  lines.push('Use this summary for questions about cheapest, budget, most expensive, or price range queries.');

  return {
    title: 'Catalog: Price Range Summary (Cheapest & Most Expensive)',
    content: lines.join('\n'),
  };
}

// ─── Seed into backend ───────────────────────────────────

async function seedCatalog(docs) {
  console.log(`\n📚 Seeding ${docs.length} catalog documents to ChromaDB...`);

  for (const doc of docs) {
    console.log(`   👉 ${doc.title}`);
    const res = await fetch(`${BACKEND_URL}/api/knowledge`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(doc),
    });

    if (!res.ok) {
      const errText = await res.text();
      console.error(`   ❌ Failed: ${errText}`);
    } else {
      const data = await res.json();
      console.log(`      ✅ Indexed (${data.chunks || 1} chunks)`);
    }
  }
}

// ─── Main ────────────────────────────────────────────────

async function main() {
  console.log('====================================================');
  console.log('🏷️  GoAnsuran Catalog Seeder — Live Product Pricing');
  console.log('====================================================');

  // Step 1: Fetch live data
  console.log('\n📡 Step 1: Fetching product catalog from api.ansuran2u.com...');
  const products = await fetchAllProducts();

  if (products.length === 0) {
    console.error('❌ No products fetched! Check network or API availability.');
    console.error('   Tip: Ensure Docker containers (api, chromadb) are running.');
    process.exit(1);
  }

  console.log(`\n   Total products fetched: ${products.length}`);

  // Quick brand summary
  const brandCount = {};
  for (const p of products) {
    const b = p.brand?.name || 'Unknown';
    brandCount[b] = (brandCount[b] || 0) + 1;
  }
  console.log('   Brand breakdown:', JSON.stringify(brandCount));

  // Step 2: Build knowledge documents
  console.log('\n📦 Step 2: Building catalog knowledge documents...');
  const docs = buildCatalogDocs(products);
  console.log(`   Generated ${docs.length} brand-grouped documents:`);
  for (const d of docs) {
    console.log(`      • ${d.title}`);
  }

  // Step 3: Seed into ChromaDB
  console.log('\n🚀 Step 3: Seeding to ChromaDB...');
  await seedCatalog(docs);

  console.log('\n====================================================');
  console.log('🎉 Catalog Seeding Complete!');
  console.log(`   ${products.length} products across ${docs.length} documents.`);
  console.log('   Ria can now quote REAL deposit & monthly amounts.');
  console.log('====================================================');
}

main().catch(err => {
  console.error('\n❌ Fatal error:', err.message);
  process.exit(1);
});
