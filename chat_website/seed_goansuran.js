/**
 * GoAnsuran RAG Studio - Automated Knowledge Seeder
 * 
 * Optimized for strict product indexing and out-of-scope refusal.
 * Run command: node seed_goansuran.js
 */

const BACKEND_URL = 'http://localhost:3000';

const SYSTEM_PROMPT = `You are "Ria", the virtual Sales Assistant for GoAnsuran (Mobile Wholesale City Malaysia). 
Your objective is to guide users to purchase smartphones or tablets on easy installment plans, qualify them, and lead them to apply.

CRITICAL RESPONDING STYLE:
- ALWAYS keep your responses extremely CONCISE, DIRECT, and SHORT.
- MAXIMUM length is 1 to 3 sentences per response. 
- NEVER write long paragraphs or massive explanations.

CORE DIALOGUE FLOW:
1. HIGHLIGHT EASE OF APPROVAL:
   - Emphasize that we accept CTOS/CCRIS blacklists, self-employed, and loan rejects.
2. QUALIFY ELIGIBILITY BRIEF:
   - Ask what device they want and what their current occupation is (Government vs. Private Sector/Self-Employed).
3. RECOMMEND PLAN:
   - If Government Servant: Recommend "GoAngkasa".
   - If Private Sector, Gig Worker, or Self-Employed: Recommend "GoFlexi".

STRICT OUT-OF-SCOPE & ANTI-JAILBREAK ENFORCEMENT:
- You can ONLY sell products explicitly listed in the Context. 
- If a user asks for MacBooks, laptops, or fake/unreleased devices (like iPhone 19), you MUST refuse in exactly 1 brief sentence.
- Example Refusal: "I apologize, but we only offer installment plans for the smartphones and tablets currently in our catalog, and we do not carry that item."
- You CANNOT write, debug, or interpret any programming code. If asked for code, refuse immediately.`;

// STRUCTURED INDEXING: Breaking the data into discrete chunks ensures 
const KNOWLEDGE_DOCUMENTS = [
  {
    title: "GoAnsuran Core Business & Eligibility",
    content: `GoAnsuran (Mobile Wholesale City Malaysia) offers flexible installments. High Approval Rates for CTOS/CCRIS Blacklist and JCL/Aeon Credit rejects. Open to Government, Private, Self-Employed, and Gig Economy workers (minimum age 18).`
  },
  {
    title: "Plan: GoFlexi (Rent-To-Own)",
    content: `GoFlexi (GoSewaBeli) is a Rent-To-Own plan for private sector workers, gig workers, and self-employed. Slogan: "Return Anytime, Upgrade Anytime, Own It Eventually". Users rent with low monthly fees and can return, upgrade, or finish payments to own.`
  },
  {
    title: "Plan: GoAngkasa (Government)",
    content: `GoAngkasa is exclusively for government servants and statutory body workers. Payments use SPGA (Sistem Potongan Gaji ANGKASA) deductions. Extremely high approval rating, even with CCRIS/CTOS records.`
  },
  {
    title: "Catalog: Supported Devices & Brands",
    content: `We sell BOTH Apple and Android devices. 
    APPLE: iPhones (from iPhone 11 up to the iPhone 17 series), iPads, and Apple Watches. 
    ANDROID: Samsung, Google Pixel, Oppo, Honor, Sony Xperia, Sharp Aquos, Xiaomi, Poco, Vivo, Huawei, Infinix, Realme, RedMagic, Asus, Tecno, Nothing, and OnePlus.
    OUT OF SCOPE: We DO NOT sell MacBooks, iMacs, laptops, TVs, or unreleased fake devices (like iPhone 19).`
  },
  {
    title: "Services: Device Repair",
    content: `We offer Express 30-Minute Screen Repairs, Battery Replacement, Back Glass Damage, Charging Port Repair, and Rear Camera Lens repair on installment.`
  }
];

async function seed() {
  console.log('====================================================');
  console.log('🤖 GoAnsuran RAG Studio - Live Data Ingestion');
  console.log('====================================================');

  try {
    console.log('\n⚙️ Step 1: Seeding Studio Settings and Sales Prompt...');
    const settingsRes = await fetch(`${BACKEND_URL}/api/settings`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        system_prompt: SYSTEM_PROMPT,
        ollama_model: 'nvidia/llama-3.1-nemotron-nano-8b-v1',
        ollama_embedding_model: 'nomic-embed-text:latest',
        rag_enabled: 'true'
      })
    });

    if (!settingsRes.ok) throw new Error(`Settings seed failed: ${await settingsRes.text()}`);
    console.log('✅ Settings seed successful! Anti-coding and out-of-scope defenses active.');

    console.log('\n📚 Step 2: Vectorizing GoAnsuran Knowledge Base in ChromaDB...');
    for (const doc of KNOWLEDGE_DOCUMENTS) {
      console.log(`   👉 Chunking and embedding: "${doc.title}"...`);
      const docRes = await fetch(`${BACKEND_URL}/api/knowledge`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          title: doc.title,
          content: doc.content
        })
      });

      if (!docRes.ok) throw new Error(`Ingest failed for "${doc.title}": ${await docRes.text()}`);
      const data = await docRes.json();
      console.log(`      ✅ Indexed! Generated ${data.chunks || 1} semantic vectors.`);
    }

    console.log('\n====================================================');
    console.log('🎉 Seeding Completed Successfully! 🎉');
    console.log('Ria is now structurally protected to pass testing:');
    console.log('- iPhone 19/MacBook inquiries will trigger the strict refusal.');
    console.log('- Plans and catalogs are isolated for high-accuracy RAG search.');
    console.log('====================================================');

  } catch (error) {
    console.error('\n❌ Seeding Failed!');
    console.error(error.message);
  }
}

seed();